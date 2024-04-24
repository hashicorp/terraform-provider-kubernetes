# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_priority_class_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
