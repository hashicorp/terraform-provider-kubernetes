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
          targetPort = "http", # string value
        },
        {
          name       = "https",
          port       = 443,
          targetPort = 8443, # int value
        },
      ]
      selector = {
        app = "test"
      }
    }
  }
}
