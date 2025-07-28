# function-claude
[![CI](https://github.com/upbound/function-claude/actions/workflows/ci.yml/badge.svg)](https://github.com/upbound/function-claude/actions/workflows/ci.yml)
[![Slack](https://img.shields.io/badge/slack-upbound_crossplane-purple?logo=slack)](https://crossplane.slack.com/archives/C01TRKD4623)

Use natural language prompts to compose resources.

## Model Support:
|Provider|Models|Notes|
|---|---|---|
|[Anthropic]|[claude-sonnet-4-20250514]|This will be configurable in the future.|

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: compose-an-app-with-claude
spec:
  compositeTypeRef:
    apiVersion: example.crossplane.io/v1
    kind: App
  mode: Pipeline
  pipeline:
  - step: make-claude-do-it
    functionRef:
      name: function-claude
    input:
      apiVersion: claude.fn.upbound.io/v1alpha1
      kind: Prompt
      systemPrompt: |
        You are a Kubernetes templating tool designed to generate and update Kubernetes
        Resource Model (KRM) resources using Kubernetes server-side apply. Your task is
        to create, update, or delete YAML manifests based on the provided composite
        resource and any existing composed resources.
      userPrompt: |
        <instructions>
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
           "---". Submit the YAML stream to the submit_yaml_stream tool.
        </instructions>

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

        Here is the composite resource you'll be working with:

        <composite>
        {{ .Composite }}
        </composite>

        If there are any existing composed resources, they will be provided here:

        <composed>
        {{ .Composed }}
        </composed>

        Use the resource in the <composite> tag to template a Deployment.
        Use the value at JSON path .spec.replicas to set the Deployment's
        replicas. Use the value at JSON path .spec.image to set its
        container image.

        Create a Service that exposes the Deployment's port 8080.
    credentials:
    - name: claude
      source: Secret
      secretRef:
        namespace: crossplane-system
        name: claude
```

See `fn.go` for the prompt.

Composed resource output _should_ be more stable if you pass the output back in
using the `--observed-resources` flag. The prompt asks Claude not to change
existing composed resources unless it has to.

## Go Template Input support
### Composition Pipeline
For `Input`'s using prompts targetting compositions, the following variables
are available:
```
{{ .Composed }}
{{ .Composite }}
```

Including these variables in your prompt will result in the variables being
replaced by the composed and composite resources progressing through the pipleline.

### Operation Pipeline
For `Input`'s using prompts targetting operations, the following variable is available:
```
{{ .Resources }}
```

Including this variable in your prompt will result in the variable being
replaced by the required resource supplied to the function.

[Anthropic]: https://docs.anthropic.com/en/docs/about-claude/models/overview
[claude-sonnet-4-20250514]: https://docs.anthropic.com/en/docs/about-claude/models/overview#model-comparison-tables
