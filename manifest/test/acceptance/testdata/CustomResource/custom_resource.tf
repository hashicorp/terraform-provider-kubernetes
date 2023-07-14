# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = var.group_version
    kind       = var.kind
    metadata = {
      namespace = var.namespace
      name      = var.name
    }
    data = "this is a test"
    refs = 98.765
    stuff = [
      {
        foo = null
      }
    ]
    limits = {
      "foo" = "bar"
      "baz" = 42
    }
  }
}
