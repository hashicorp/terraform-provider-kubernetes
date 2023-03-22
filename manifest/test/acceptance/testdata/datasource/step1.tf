# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_config" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = var.name
      "namespace" = var.namespace
    }
    "data" = {
      "TEST" = "hello world"
    }
  }
}
