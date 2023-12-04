# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Secret"
    metadata = {
      name      = var.name
      namespace = var.namespace

      annotations = {
        "kubernetes.io/service-account.name" = "default"
      }
    }
    type = "kubernetes.io/service-account-token"
  }

  wait {
    fields = {
      "metadata.annotations" = "^.*$",
    }
  }

  timeouts {
    create = "10s"
  }
}
