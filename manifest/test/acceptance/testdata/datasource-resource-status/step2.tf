data "kubernetes_resource" "test_deploy" {
  api_version = "v1"
  kind = "Deployment"
  metadata {
    name = var.name
    namespace = var.namespace
  }
}
