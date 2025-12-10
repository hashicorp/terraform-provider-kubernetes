# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "configmap" {

  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "extraconfig-from-values.hcl" = <<EOT
          disable_mlock = true
          ui = true

          listener "tcp" {
            tls_disable = 1
            address = "[::]:8200"
            cluster_address = "[::]:8201"
          }
          storage "file" {
            path = "/vault/data"
          }
      EOT
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault"
      }
      "name"      = "${var.name}-vault-config"
      "namespace" = var.namespace
    }
  }
}
