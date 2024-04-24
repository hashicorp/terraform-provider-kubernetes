# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_secret" "example" {
  metadata {
    name      = "example-secret"
    namespace = "kube-system"
  }
  binary_data = {
    "keystore.p12" = ""
    another_field  = ""
  }
}
