# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_deploy" {
  manifest = {
    "apiVersion" = "apps/v1"
    "kind"       = "Deployment"
    "metadata" = {
      "name" = var.name
      "namespace" = var.namespace
    }
    "spec" = {
      "selector" = {
        "matchLabels" = {
          "test" = "MyExampleApp"
        }
      }

      "template" = {
        "metadata" = {
          "labels" = {
            "test" = "MyExampleApp"
          }
        }

        "spec" = {
          "containers" = [
            {
               "image" = "nginx:1.21.6"
               "name"  = "example"
            }
          ]
        }
      }
    }
  }
}
