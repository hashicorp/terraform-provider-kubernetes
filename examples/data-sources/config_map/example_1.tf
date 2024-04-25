# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_config_map" "example" {
  metadata {
    name = "my-config"
  }
}
