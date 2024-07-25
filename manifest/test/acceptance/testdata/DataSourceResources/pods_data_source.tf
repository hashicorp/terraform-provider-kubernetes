# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_resources" "pods" {
  kind        = "Pod"
  api_version = "v1"
  namespace   = var.namespace
}
