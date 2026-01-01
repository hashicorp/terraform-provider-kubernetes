# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_resources" "pods" {
  kind        = "Pod"
  api_version = "v1"
  namespace   = var.namespace
}
