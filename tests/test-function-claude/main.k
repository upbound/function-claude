import models.io.upbound.dev.meta.v1alpha1 as metav1alpha1

_items = [
    metav1alpha1.CompositionTest{
        metadata.name: "function-claude"
        spec= {
            functionCredentialsPath: "example/secret.yaml"
            assertResources: [
                {
                    apiVersion: "example.crossplane.io/v1"
                    kind: "App"
                    metadata: {
                        name: "example-db"
                    }
                }
                {
                    apiVersion: "apps/v1"
                    kind: "Deployment"
                    metadata: {
                      annotations: {
                        "crossplane.io/composition-resource-name": "deployment"
                        "upbound.io/name": "deployment"
                      }
                      generateName: "example-db-"
                      labels: {
                        "crossplane.io/composite": "example-db"
                      }
                      ownerReferences: [{
                        apiVersion: "example.crossplane.io/v1"
                        blockOwnerDeletion: True
                        controller: True
                        kind: "App"
                        name: "example-db"
                        uid: ""
                      }]
                    }
                    spec: {
                      replicas: 10
                      selector: {
                        matchLabels: {
                          app: "example-db"
                        }
                      }
                      template: {
                        metadata: {
                          labels: {
                            app: "example-db"
                          }
                        }
                        spec: {
                          containers: [{
                            image: "upbound/example:v1"
                            name: "example-db"
                          }]
                        }
                      }
                    }
                }
                {
                    apiVersion: "v1"
                    kind: "Service"
                    metadata: {
                      annotations: {
                        "crossplane.io/composition-resource-name": "service"
                        "upbound.io/name": "service"
                      }
                      generateName: "example-db-"
                      labels: {
                        "crossplane.io/composite": "example-db"
                      }
                      ownerReferences: [{
                        apiVersion: "example.crossplane.io/v1"
                        blockOwnerDeletion: True
                        controller: True
                        kind: "App"
                        name: "example-db"
                        uid: ""
                      }]
                    }
                    spec: {
                      ports: [{
                        port: 8080
                        protocol: "TCP"
                        targetPort: 8080
                      }]
                      selector: {
                        app: "example-db"
                      }
                    }
                }
            ]
            compositionPath: "example/composition.yaml"
            xrPath: "example/xr.yaml"
            timeoutSeconds: 60
            validate: False
        }
    }
]
items= _items