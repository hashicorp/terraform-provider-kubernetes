resource "kubernetes_manifest" "test_config" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = var.name
      "namespace" = var.namespace
    }
    "data" = {
      "TEST" = "hello world"
    }
  }
}
