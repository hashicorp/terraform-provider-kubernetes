provider "kubernetes-alpha" {
}

resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "v1"
    kind       = "Secret"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    data = {
      PGUSER     = "username"
      PGPASSWORD = "password"
    }
  }
}
