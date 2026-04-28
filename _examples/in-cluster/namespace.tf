# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_namespace_v1" "this" {
  metadata {
    name = "this"
  }
}
