# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Regression shape from PR #2822 issue comment:
# - mapdata.test.datas has a concrete list(string)
# - mapdata.test2 omits datas entirely
#
# Without map(dynamic) normalization, this can panic Terraform during plan with:
# "panic: inconsistent map element types".

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = var.group_version
    kind       = var.kind
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      mapdata = {
        test = {
          datas = ["10.10.0.0/16", "10.20.0.0/16"]
        }
        test2 = {}
      }
    }
  }
}
