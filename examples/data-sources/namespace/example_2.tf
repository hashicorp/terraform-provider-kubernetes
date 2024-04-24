# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_namespace" "example" {
  metadata {
    name = "kube-system"
  }
}
