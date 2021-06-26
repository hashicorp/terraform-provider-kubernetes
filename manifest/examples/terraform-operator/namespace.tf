resource "kubernetes_manifest" "namespace" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name = var.namespace
    }
  }
}

