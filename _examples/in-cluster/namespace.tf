# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_namespace_v1" "this" {
  metadata {
    name = "this"
  }
}
