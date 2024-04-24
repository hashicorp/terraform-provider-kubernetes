# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_labels" "example" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = "my-config"
  }
  labels = {
    "owner" = "myteam"
  }
}
