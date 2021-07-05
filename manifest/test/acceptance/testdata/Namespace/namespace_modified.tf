provider "kubernetes-alpha" {
}

resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name      = var.name
      labels = {
        test = "test"
      }
    }
  }
}
