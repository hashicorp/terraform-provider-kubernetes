resource "kubernetes_manifest_v2" "config" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = "app-config"
      namespace = "default"
    }
    data = {
      LOG_LEVEL = "info"
    }
  }
}
