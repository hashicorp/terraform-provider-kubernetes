resource "kubernetes_manifest" "test_config" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = var.name
      "namespace" = var.namespace
      "labels" = {
        "TEST" = "terraform"
      }
    }
    "data" = {
      "TEST" = "hello world"
    }
  }
}
