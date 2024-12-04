resource "kubernetes_stateful_set_v1_gen" {
  package = "appsv1"

  api_version = "apps/v1"
  kind        = "StatefulSet"

  description = "statefulset"

  output_filename_prefix = "statefulset"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/apis__apps__v1_openapi.json"
    create_path = "/apis/apps/v1/namespaces/{namespace}/statefulsets"
    read_path   = "/apis/apps/v1/namespaces/{namespace}/statefulsets/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true
    
    autocrud_options {
      wait_for_deletion = true
    
      # TODO after create hook for waiting for rollout
      # TODO after update hook for waiting for rollout
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

  # custom_attributes = [
  #   "wait_for_rollout"
  # ]

  required_attributes = [
    "metadata",
    "spec"
  ]

  computed_attributes = [
    "metadata.uid",
    "metadata.resource_version",
    "metadata.generation",
    "metadata.name",
    "metadata.namespace"
  ]

  immutable_attributes = [
    "metadata.name"
  ]
}
