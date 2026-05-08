# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_role_binding" "tfc-role-binding" {
  metadata {
    name      = "${kubernetes_manifest.namespace.object.metadata.name}-sync-workspace"
    namespace = kubernetes_manifest.namespace.object.metadata.name
    labels = {
      app = kubernetes_manifest.namespace.object.metadata.name
    }
  }
  role_ref {
    kind      = "Role"
    name      = kubernetes_role.tfc-role.metadata[0].name
    api_group = "rbac.authorization.k8s.io"
  }
  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.tfc-service-account.metadata[0].name
    namespace = kubernetes_manifest.namespace.object.metadata.name
  }
}
