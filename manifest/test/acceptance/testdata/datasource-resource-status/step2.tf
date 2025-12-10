# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_resource" "test_deploy" {
  api_version = "apps/v1"
  kind = "Deployment"
  metadata {
    name = var.name
    namespace = var.namespace
  }
}
