# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_secret" "tfc-api-token" {
  metadata {
    name      = "terraformrc"
    namespace = kubernetes_manifest.namespace.object.metadata.name
    labels = {
      app = kubernetes_manifest.namespace.object.metadata.name
    }
  }

  data = {
    credentials = var.tfc_credentials
  }
}
