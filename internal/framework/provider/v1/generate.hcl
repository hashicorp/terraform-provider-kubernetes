resource "kubernetes_secret_v1_gen" {
  package = "v1"

  api_version = "v1"
  kind        = "Secret"

  description = "provides mechanisms to inject containers with sensitive information"

  output_filename_prefix = "secret"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    create_path = "/api/v1/namespaces/{namespace}/secrets"
    read_path   = "/api/v1/namespaces/{namespace}/secrets/{name}"
  }

  generate {
    schema     = true
    model      = true
    autocrud = true
  }
  ignore_attributes = ["metadata.managed_fields"]
}