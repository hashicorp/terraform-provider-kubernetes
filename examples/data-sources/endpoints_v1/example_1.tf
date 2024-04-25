# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_endpoints_v1" "api_endpoints" {
  metadata {
    name      = "kubernetes"
    namespace = "default"
  }
}
