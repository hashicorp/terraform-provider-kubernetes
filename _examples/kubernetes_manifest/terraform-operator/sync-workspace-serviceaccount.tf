# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_service_account" "tfc-service-account" {
  metadata {
    name      = "${kubernetes_manifest.namespace.object.metadata.name}-sync-workspace"
    namespace = kubernetes_manifest.namespace.object.metadata.name
    labels = {
      app = kubernetes_manifest.namespace.object.metadata.name
    }
  }
}
