package main

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/crossplane/function-template-go/input/v1beta1"
)

const (
	credName = "claude"
	credKey  = "ANTHROPIC_API_KEY"
)

const prompt = `
You are a Kubernetes templating tool designed to generate and update Kubernetes Resource Model (KRM) resources using Kubernetes server-side apply. Your task is to create or modify YAML manifests based on the provided composite resource and any existing composed resources.

Here is the composite resource you'll be working with:

<composite>
{{ .Composite }}
</composite>

If there are any existing composed resources, they will be provided here:

<composed>
{{ .Composed }}
</composed>

Additional input will be provided here:

<input>
{{ .Input }}
</input>

Please follow these instructions carefully:

1. Analyze the provided composite resource and any existing composed resources.

2. Generate a stream of YAML manifests based on the composite resource. Each manifest should:
   a. Be valid for Kubernetes server-side apply (fully specified intent).
   b. Omit names and namespaces.
   c. Include an annotation with the key "upbound.io/name". The value should be the name of the resource in the <composite> tag appended with the kind of the templated resource. If there are multiple resources of the same kind, append sequential numbers to differentiate them.
   d. Use labels to create relationships between resources when necessary. Use the name of the resource in the <composite> tag for these labels.

3. If existing composed resources are provided, try to reuse their values as much as possible. Only change values when absolutely necessary.

4. The output should be a stream of YAML manifests, each separated by "---". The output must be in <output> tags.

Before generating the YAML manifests, use <analysis> tags to analyze the input and plan your approach. In your analysis:

a. List all resources mentioned in the composite resource.
b. Compare with existing composed resources (if any).
c. Plan the necessary actions (create, update, or reuse) for each resource.
d. Outline how to ensure proper annotations and labels for each resource.
e. Consider any additional input provided in the <input> tag.

After your analysis, provide the YAML stream as your final output.

Example output structure (generic, for illustration purposes only):

<analysis>
[Your structured analysis here]
</analysis>

<output>
apiVersion: [api-version]
kind: [resource-kind]
metadata:
  annotations:
    upbound.io/name: [composite-name-resource-kind]
  labels:
    [relationship-labels-if-needed]
spec:
  [resource-specific-fields]
---
[Additional resources as needed]
</output>

Please proceed with your analysis and YAML generation.
`

// Variables used to form the prompt.
type Variables struct {
	// Observed composite resource, as a YAML manifest.
	Composite string

	// Observed composed resources, as a stream of YAML manifests.
	Composed string

	// Input - i.e. user prompt.
	Input string
}

// Function asks Claude to compose resources.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	prompt *template.Template
	output *regexp.Regexp

	log logging.Logger
}

// NewFunction creates a new function powered by Claude.
func NewFunction(log logging.Logger) *Function {
	return &Function{
		log:    log,
		prompt: template.Must(template.New("prompt").Parse(prompt)),

		// The ?s flag makes .* match across newlines in that group.
		// Flag groups can't be capture groups, so there's a nested
		// capture group.
		output: regexp.MustCompile(`<output>(?s:(.*))</output>`),
	}
}

// RunFunction runs the Function.
func (f *Function) RunFunction(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) { //nolint:gocyclo // TODO(negz): Factor out the API calling bits.
	log := f.log.WithValues("tag", req.GetMeta().GetTag())
	log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1beta1.Prompt{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	c, err := request.GetCredentials(req, credName)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Anthropic API key from credential %q", credName))
		return rsp, nil
	}
	if c.Type != resource.CredentialsTypeData {
		response.Fatal(rsp, errors.Errorf("expected credential %q to be %q, got %q", credName, resource.CredentialsTypeData, c.Type))
		return rsp, nil
	}
	b, ok := c.Data[credKey]
	if !ok {
		response.Fatal(rsp, errors.Errorf("credential %q is missing required key %q", credName, credKey))
		return rsp, nil
	}

	// TODO(negz): Where the heck is the newline at the end of this key
	// coming from? Bug in crossplane render?
	key := strings.Trim(string(b), "\n")

	// TODO(negz): I'm using YAML as input/output because I assume the model
	// will be better able to represent Kubernetes stuff as YAML manifests
	// than as e.g. JSON. YAML's much more prevalent in examples etc. Could
	// be worth validating this - could we use JSON instead to skip extra
	// conversion?
	xr, err := CompositeToYAML(req.GetObserved().GetComposite())
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot convert observed XR to YAML"))
		return rsp, nil
	}

	cds, err := ComposedToYAML(req.GetObserved().GetResources())
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot convert observed composed resources to YAML"))
		return rsp, nil
	}

	prompt := &strings.Builder{}
	if err := f.prompt.Execute(prompt, &Variables{Composite: xr, Composed: cds, Input: in.Prompt}); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot build prompt from template"))
		return rsp, nil
	}

	log.Debug("Using prompt", "prompt", prompt.String())

	client := anthropic.NewClient(option.WithAPIKey(key))
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 1024,
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		// TODO(negz): Use a system prompt? The prompt improver
		// recommended rolling it into the user prompt.
		Temperature: param.Opt[float64]{Value: 0}, // As little randomness as possible.
		Messages: []anthropic.MessageParam{{
			Role:    anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{{OfText: &anthropic.TextBlockParam{Text: prompt.String()}}},
		}},
	})
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot message Claude"))
		return rsp, nil
	}

	if len(message.Content) != 1 {
		response.Fatal(rsp, errors.Errorf("expected 1 response, got %d", len(message.Content)))
		return rsp, nil
	}
	content := message.Content[0]
	if content.Type != "text" {
		response.Fatal(rsp, errors.Errorf("expected text response, got %q", content.Type))
		return rsp, nil
	}
	log.Debug("Got content from Claude", "content", content.Text)

	// This should be a YAML stream.
	matches := f.output.FindStringSubmatch(content.Text)
	if len(matches) != 2 {
		response.Fatal(rsp, errors.Errorf("expected 1 match in response for regular expression %q, got %d", f.output.String(), len(matches)))
		return rsp, nil
	}
	output := matches[1]
	log.Debug("Extracted output from content", "output", output)

	dcds, err := ComposedFromYAML(output)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot parse Claude output as YAML"))
		return rsp, nil
	}
	rsp.Desired.Resources = dcds

	// TODO(negz): Support setting XR status fields too.

	return rsp, nil
}

// CompositeToYAML returns the XR as YAML.
func CompositeToYAML(xr *fnv1.Resource) (string, error) {
	j, err := protojson.Marshal(xr.GetResource())
	if err != nil {
		return "", errors.Wrap(err, "cannot convert XR to JSON")
	}
	y, err := yaml.JSONToYAML(j)
	return string(y), errors.Wrap(err, "cannot convert XR to YAML")
}

// ComposedToYAML returns the supplied composed resources as a YAML stream. The
// resources are annotated with their upbound.io/name annotations.
func ComposedToYAML(cds map[string]*fnv1.Resource) (string, error) {
	// TODO(negz): Does giving the model stable input like this increase the
	// likelihood it'll be able to match resources correctly?
	keys := make([]string, 0, len(cds))
	for k := range cds {
		keys = append(keys, k)
	}
	sort.StringSlice(keys).Sort()

	composed := &strings.Builder{}

	for _, name := range keys {
		ocd := cds[name]
		jocd, err := protojson.Marshal(ocd.GetResource())
		if err != nil {
			return "", errors.Wrap(err, "cannot convert composed resource to JSON")
		}

		jocd, err = sjson.SetBytes(jocd, "metadata.annotations.upbound\\.io/name", name)
		if err != nil {
			return "", errors.Wrapf(err, "cannot set upbound.io/name annotation")
		}

		yocd, err := yaml.JSONToYAML(jocd)
		if err != nil {
			return "", errors.Wrap(err, "cannot convert composed resource to YAML")
		}
		composed.WriteString("---\n")
		composed.Write(yocd)
	}

	return composed.String(), nil
}

// ComposedFromYAML parses the supplied YAML stream as desired composed
// resources. The resource names are extracted from the upbound.io/name
// annotation.
func ComposedFromYAML(y string) (map[string]*fnv1.Resource, error) {
	out := make(map[string]*fnv1.Resource)

	for _, doc := range strings.Split(y, "---") {
		j, err := yaml.YAMLToJSON([]byte(doc))
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse YAML")
		}

		s := &structpb.Struct{}
		if err := protojson.Unmarshal(j, s); err != nil {
			return nil, errors.Wrap(err, "cannot parse JSON")
		}

		name := gjson.GetBytes(j, "metadata.annotations.upbound\\.io/name").String()
		out[name] = &fnv1.Resource{Resource: s}
	}

	return out, nil
}
