data "kubernetes_endpoints_v1" "api_endpoints" {
  metadata {
    name      = "kubernetes"
    namespace = "default"
  }
}
