# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_priority_class" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
