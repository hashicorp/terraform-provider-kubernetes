# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "test-role" {

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
