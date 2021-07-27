provider "kubernetes-alpha" {
}

resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = var.group_version
    kind       = var.kind
    metadata = {
      namespace = var.namespace
      name      = var.name
    }
    data = "this is a test"
  }
}
