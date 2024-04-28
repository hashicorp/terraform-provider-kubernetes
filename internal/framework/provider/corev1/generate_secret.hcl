resource "kubernetes_secret_v1_gen" {
  package = "corev1"

  api_version = "v1"
  kind        = "Secret"

  description = "secrets store secret information for pods"

  output_filename_prefix = "secret"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    create_path = "/api/v1/namespaces/{namespace}/secrets"
    read_path   = "/api/v1/namespaces/{namespace}/secrets/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true
    
    autocrud_options {
      hooks {
        before {
          create = true
          update = true
        }
        after {
          create = true
          update = true
          read   = true
        }
      }
    }
  }

  ignored_attributes = [
    "api_version",
    "kind",
    "metadata.finalizers",
    "metadata.managed_fields",
    "metadata.owner_references",
    "metadata.self_link",
    "metadata.creation_timestamp",
    "metadata.deletion_timestamp",
    "metadata.deletion_grace_period_seconds",
  ]

  required_attributes = [
    "metadata"
  ]

  computed_attributes = [
    "metadata.uid",
    "metadata.resource_version",
    "metadata.generation",
    "metadata.name",
  ]

  sensitive_attributes = [
    "data"
  ]

  immutable_attributes = [
    "metadata.name",
    "metadata.namespace"
  ]
}
