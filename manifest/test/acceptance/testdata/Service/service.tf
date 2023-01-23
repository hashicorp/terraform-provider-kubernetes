resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      ports = [
        {
          name       = "http",
          port       = 80,
          targetPort = "http",
        }
      ]
      selector = {
        app = "test"
      }
    }
  }
}
