# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0


variable "name" {
  default = "test-service"
}

variable "namespace" {
  default = "default"
}
resource "kubernetes_manifest" "service-injector" {

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
          "name"       = "http"
          "port"       = 80
          "targetPort" = 8080
          "protocol"   = "TCP"
        },
        {
          "name"       = "https"
          "port"       = 443
          "targetPort" = "https"
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
