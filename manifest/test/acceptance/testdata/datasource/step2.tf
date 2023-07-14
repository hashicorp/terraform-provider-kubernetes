# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_resource" "test_config" {
  api_version = "v1"
  kind = "ConfigMap"
  metadata {
    name = var.name
    namespace = var.namespace
  }
}

resource "kubernetes_manifest" "test_config2" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = var.name2
      "namespace" = var.namespace
    }
    "data" = {
      "TEST" = data.kubernetes_resource.test_config.object.data.TEST
    }
  }
}
