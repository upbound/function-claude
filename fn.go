package main

import (
	"context"
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

const system = `
You are a Kubernetes templating tool designed to generate and update Kubernetes
Resource Model (KRM) resources using Kubernetes server-side apply. Your task is
to create, update, or delete YAML manifests based on the provided composite
resource and any existing composed resources.
`

const prompt = `
Here is the composite resource you'll be working with:

<composite>
{{ .Composite }}
</composite>

If there are any existing composed resources, they will be provided here:

<composed>
{{ .Composed }}
</composed>

Additional input is provided here:

<input>
{{ .Input }}
</input>

Please follow these instructions carefully:

1. Analyze the provided composite resource and any existing composed resources.

2. Analyze the input to understand what composed resources you should create,
   update, or delete. You may be asked to derive composed resources from the
   composite resource, or from other composed resources.

3. Generate a stream of YAML manifests based on your analysis in steps 1 and 2.
   Each manifest should:
   a. Be valid for Kubernetes server-side apply (fully specified intent).
   b. Omit names and namespaces.
   c. Include an annotation with the key "upbound.io/name". This annotation
      must uniquely identify the manifest within the YAML stream. It must be
      lowercase, hyphen separated, and less than 30 characters long. Prefer
      to use the manifest's kind. If two or more manifests have the same
      kind, look for something unique about the manifest and append that to
      the kind. This annotation is used to match the manifests you return to
      any manifests that were passed you inside the <composed> tag, so if
      your intent is to update a manifest never change its "upbound.io/name"
      annotation. This is critically important.
   d. If it's necessary to use labels to create relationships between
      resources, use the name of the composite resource as the label value.

4. If there are existing composed resources:
    a. You can update an existing composed resource by including it in your
       output with any changes you deem necessary based on the input. Try to
       reuse existing composed resource values as much as possible. Only
       change values when you're sure it's necessary.
    b. If the input indicates that a resource is no longer required, you can
       delete it by omitting it from your output.

5. Your output must only be a stream of YAML manifests, each separated by
   "---". The output must be parseable using a YAML parser.

Example output structure:

<example>
---
apiVersion: [api-version]
kind: [resource-kind]
metadata:
  annotations:
    upbound.io/name: [resource-kind]
  labels:
    [relationship-labels-if-needed]
spec:
  [resource-specific-fields]
---
[Additional resources as needed]
</example>
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
	log    logging.Logger
}

// NewFunction creates a new function powered by Claude.
func NewFunction(log logging.Logger) *Function {
	return &Function{
		log:    log,
		prompt: template.Must(template.New("prompt").Parse(prompt)),
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
		MaxTokens:   1024,
		Model:       anthropic.ModelClaudeSonnet4_0,
		System:      []anthropic.TextBlockParam{{Text: system}},
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

	dcds, err := ComposedFromYAML(content.Text)
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
	composed := &strings.Builder{}

	for name, ocd := range cds {
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
		if doc == "" {
			continue
		}
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
