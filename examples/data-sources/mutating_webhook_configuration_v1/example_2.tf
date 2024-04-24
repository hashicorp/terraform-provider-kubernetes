# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_mutating_webhook_configuration_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}
