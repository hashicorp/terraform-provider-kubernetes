resource "kubernetes_service_v1_gen" {
  package = "corev1"

  api_version = "v1"
  kind        = "Service"

  description = "Service is an abstraction which defines a logical set of pods and a policy by which to access them"

  output_filename_prefix = "service"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    create_path = "/api/v1/namespaces/{namespace}/services"
    read_path   = "/api/v1/namespaces/{namespace}/services/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true
    
    autocrud_options {
      wait_for_deletion = true
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
