# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name      = var.name
      namespace = var.namespace
      annotations = {
        test = "1"
      }
      labels = {
        test = "2"
      }
    }
    spec = {
      ports = [
        {
          name       = "http",
          port       = 80,
          targetPort = "http", # string value
        },
        {
          name       = "https",
          port       = 443,
          targetPort = 8443, # numerical value
        }
      ]
      selector = {
        app = "test"
      }
    }
  }
}
