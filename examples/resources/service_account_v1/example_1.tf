# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_service_account_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "kubernetes_secret_v1" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = kubernetes_service_account_v1.example.metadata.0.name
    }

    generate_name = "terraform-example-"
  }

  type                           = "kubernetes.io/service-account-token"
  wait_for_service_account_token = true
}
