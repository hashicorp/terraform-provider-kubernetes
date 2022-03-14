resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = var.name
      namespace = var.namespace
      annotations = {
        test = "1"
      }
      labels = {
        test = "2"
      }
    }
    data = {
      foo  = "bar"
      fizz = "buzz"
    }
  }
}
