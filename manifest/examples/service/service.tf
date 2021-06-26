provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
}

variable "name" {
  default = "test-service"
}

variable "namespace" {
  default = "default"
}

resource "kubernetes_manifest" "service-injector" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Service"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
      }
      "name"      = "${var.name}-vault-agent-injector-svc"
      "namespace" = var.namespace
    }
    "spec" = {
      "ports" = [
        {
          "port"       = 443
          "targetPort" = 80
          "protocol"   = "TCP"
        },
      ]
      "selector" = {
        "app.kubernetes.io/instance" = var.name
        "app.kubernetes.io/name"     = "vault-agent-injector"
        "component"                  = "webhook"
      }
    }
  }
}
