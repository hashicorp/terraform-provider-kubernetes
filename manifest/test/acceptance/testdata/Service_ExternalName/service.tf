# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      selector = {
        app = "test"
      }
      type         = "ExternalName"
      externalName = "terraform.kubernetes.test.com"
    }
  }
}
