# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_annotations" "example" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = "my-config"
  }
  annotations = {
    "owner" = "myteam"
  }
}
