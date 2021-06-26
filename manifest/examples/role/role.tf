provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
}


resource "kubernetes_manifest" "test-role" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "rbac.authorization.k8s.io/v1"
    "kind"       = "Role"
    "metadata" = {
      "name"      = "pod-reader"
      "namespace" = "default"
      "labels" = {
        "app"         = "test-app"
        "environment" = "production"
      }
    }
    "rules" = [
      {
        "apiGroups" = [
          "",
        ]
        "resources" = [
          "pods",
        ]
        "verbs" = [
          "get",
          "list",
          "watch",
        ]
      },
    ]
  }
}
