# function-claude
[![CI](https://github.com/upbound/function-claude/actions/workflows/ci.yml/badge.svg)](https://github.com/upbound/function-claude/actions/workflows/ci.yml)

Use natural language prompts to compose resources.

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
      apiVersion: claude.fn.crossplane.io/v1beta1
      kind: Prompt
      prompt: |
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
