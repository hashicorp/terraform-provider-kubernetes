resource "kubernetes_namespace_v1_gen" {
  package = "corev1"

  api_version = "v1"
  kind        = "Namespace"

  description = "namespace"

  output_filename_prefix = "namespace"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    create_path = "/api/v1/namespaces"
    read_path   = "/api/v1/namespaces/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true
    
    autocrud_options {
      wait_for_deletion = true
      hooks {
        after {
          create = true
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
    "status",
    "spec"
  ]

  custom_attributes = [
    "wait_for_default_service_account"
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

  immutable_attributes = [
    "metadata.name"
  ]
}
