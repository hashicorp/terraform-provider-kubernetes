# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_namespace_v1" "example" {
  metadata {
    name = "kube-system"
  }
}
