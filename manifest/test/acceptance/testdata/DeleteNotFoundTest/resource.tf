resource "kubernetes_manifest" "test" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name"      = var.name
      "namespace" = var.namespace
    }
    "data" = {
      "foo" = "bar"
    }
  }
}

