resource "kubernetes_cluster_role_binding_v1_gen" {
  package = "rbacv1"

  api_version = "rbac.authorization.k8s.io/v1"
  kind        = "ClusterRoleBinding"

  description = "cluster role bindings are used to grant permissions at the cluster level"

  output_filename_prefix = "cluster_role_binding"

  openapi {
    filename    = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/apis__rbac.authorization.k8s.io__v1_openapi.json"
    create_path = "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings"
    read_path   = "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/{name}"
  }
  
  generate {
    schema     = true
    model      = true
    autocrud   = true 
  }

  ignored_attributes = [
    "api_version",
    "kind",
    "metadata.namespace",
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
    "metadata.name"
  ]
}
