# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_annotations" "example" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = "my-config"
  }
  # These annotations will be applied to the Deployment resource itself
  annotations = {
    "owner" = "myteam"
  }
  # These annotations will be applied to the Pods created by the Deployment
  template_annotations = {
    "owner" = "myteam"
  }
}
