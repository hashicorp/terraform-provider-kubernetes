resource "kubernetes_service_account_v1_gen" {
  package = "corev1"

  api_version = "v1"
  kind        = "ServiceAccount"

  description = "service accounts provide an identity for processes that run in a Pod"

  output_filename_prefix = "service_account"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    create_path = "/api/v1/namespaces/{namespace}/serviceaccounts"
    read_path   = "/api/v1/namespaces/{namespace}/serviceaccounts/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true  
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

  immutable_attributes = [
    "metadata.name",
    "metadata.namespace"
  ]
}
