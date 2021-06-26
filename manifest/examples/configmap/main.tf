provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "test-configmap" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name"      = "test-config"
      "namespace" = "default"
      "labels" = {
        "app"         = "test-app"
        "environment" = "production"
      }
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
