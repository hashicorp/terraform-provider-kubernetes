variable "cluster_name" {
  type = string
}

resource "kubernetes_manifest" "test-cfm" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test-cf"
      "namespace" = "default"
      "labels" = {
        "parent_cluster" = var.cluster_name
      }
    }
    "data" = {
      "parent_cluster" = var.cluster_name
    }
  }
}
