# The arguments stay as they are, and the first plan shows no changes.
moved {
  from = kubernetes_manifest.config
  to   = kubernetes_manifest_v2.config
}

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
