# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "annotations" = {
        "tf-k8s-acc" = "true"
      }
      "name"      = var.name
      "namespace" = var.namespace
    }
  }
}
