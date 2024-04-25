# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_secret" "example" {
  metadata {
    name = "basic-auth"
  }
}
