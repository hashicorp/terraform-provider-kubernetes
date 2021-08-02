resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name = var.name
    }
  }
}
