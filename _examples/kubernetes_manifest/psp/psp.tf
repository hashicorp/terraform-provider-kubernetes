# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


# PodSecurityPolicy only works on Kubernetes 1.17+
resource "kubernetes_manifest" "psp" {

  manifest = {
    "apiVersion" = "policy/v1beta1"
    "kind"       = "PodSecurityPolicy"
    "metadata" = {
      "name" = "example"
    }
    "spec" = {
      "fsGroup" = {
        "rule" = "RunAsAny"
      }
      "runAsUser" = {
        "rule" = "RunAsAny"
      }
      "seLinux" = {
        "rule" = "RunAsAny"
      }
      "supplementalGroups" = {
        "rule" = "RunAsAny"
      }
      "volumes" = [
        "*",
      ]
    }
  }
}
