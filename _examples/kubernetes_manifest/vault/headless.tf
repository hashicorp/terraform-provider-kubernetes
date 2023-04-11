# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "service-headless" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Service"
    "metadata" = {
      "annotations" = merge({
        "service.alpha.kubernetes.io/tolerate-unready-endpoints" = "true"
      }, var.server_service.annotations)
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault"
      }
      "name"      = "${var.name}-vault-internal"
      "namespace" = var.namespace
    }
    "spec" = {
      "clusterIP" = "None"
      "ports" = [
        {
          "name"       = "http"
          "port"       = var.server_service.port
          "targetPort" = var.server_service.targetPort
          "protocol"   = "TCP"
        },
        {
          "name"       = "https-internal"
          "port"       = 8201
          "targetPort" = 8201
          "protocol"   = "TCP"
        },
      ]
      "publishNotReadyAddresses" = true
      "selector" = {
        "app.kubernetes.io/instance" = var.name
        "app.kubernetes.io/name"     = "vault"
        "component"                  = "server"
      }
    }
  }
}
