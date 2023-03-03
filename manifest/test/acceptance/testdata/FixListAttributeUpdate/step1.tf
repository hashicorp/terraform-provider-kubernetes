# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Reported in https://github.com/hashicorp/terraform-provider-kubernetes-alpha/issues/251
#
resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata = {
      name      = var.name
      namespace = var.namespace
      annotations = {
        "deployment.kubernetes.io/revision" = "1"
      }
    }
    spec = {
      selector = {
        matchLabels = {
          app = "example"
        }
      }
      template = {
        metadata = {
          labels = {
            app = "example"
          }
        }
        spec = {
          containers = [
            {
              image   = "alpine:latest"
              name    = "ping"
              command = ["sh", "-c"]
              args    = ["ping goo.gl"]
            }
          ]
        }
      }
    }
  }
}
