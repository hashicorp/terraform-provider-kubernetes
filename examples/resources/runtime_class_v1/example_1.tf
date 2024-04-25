# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_runtime_class_v1" "example" {
  metadata {
    name = "myclass"
  }
  handler = "abcdeagh"
}
