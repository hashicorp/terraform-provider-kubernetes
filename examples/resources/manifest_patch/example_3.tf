# A setting in a ConfigMap that another tool created.
resource "kubernetes_manifest_patch" "app_settings" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = "app-settings"
      namespace = "apps"
    }
    data = {
      "log.level" = "debug"
    }
  }
}

# A label of a namespace that the cluster created.
resource "kubernetes_manifest_patch" "default_namespace" {
  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name   = "default"
      labels = { "pod-security.kubernetes.io/enforce" = "baseline" }
    }
  }
}
