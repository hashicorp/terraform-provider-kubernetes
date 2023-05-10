# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name      = var.name
      namespace = var.namespace
      annotations = {
        test = "1"
      }
      labels = {
        test = "2"
      }
    }
    spec = {
      selector = {
        app = "test"
      }
      type         = "ExternalName"
      externalName = "kubernetes-alpha.terraform.test.com"
    }
  }
}
