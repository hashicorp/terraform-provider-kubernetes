
resource "kubernetes_secret_v1_gen" {
  package = "corev1"

  api_version = "v1"
  kind        = "Secret"

  description = "configmaps store information for pods"

  output_filename_prefix = "secret"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    create_path = "/api/v1/namespaces/{namespace}/secrets"
    read_path   = "/api/v1/namespaces/{namespace}/secrets/{name}"
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
    "name",
    "namespace",
    "pretty"
  ]

computed_attributes = [
    "metadata.uid",
    "metadata.resource_version",
    "metadata.generation",
    "metadata.name",
    "type"
]

// default_values = {
//     "id" = "testing"
// }

  generate {
    schema     = true
    model      = true
    autocrud = true
    autocrud_options {
        hooks{
            before{
                create = true
            }
            after{

            }
        }
    }
  }
}
