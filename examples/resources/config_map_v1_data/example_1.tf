# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_config_map_v1_data" "example" {
  metadata {
    name = "my-config"
  }
  data = {
    "owner" = "myteam"
  }
}
