resource "kubernetes_job_v1_gen" {
  package = "batchv1"

  api_version = "batch/v1"
  kind        = "Job"

  description = "job"

  output_filename_prefix = "job"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/apis__batch__v1_openapi.json"
    create_path = "/apis/batch/v1/namespaces/{namespace}/jobs"
    read_path   = "/apis/batch/v1/namespaces/{namespace}/jobs/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true
    
    autocrud_options {
      wait_for_deletion = true
    
      # TODO after create hook for waiting for job completion
      # TODO after update hook for waiting for job completion
      # TODO after read hook for ignoring jobs with TTL = 0 
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

  # TODO:
  # custom_attributes = [
  #   "wait_for_completion"
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
  ]

  immutable_attributes = [
    "metadata.name",
    
    # job should be recreated if the spec changes
    "spec"
  ]
}
