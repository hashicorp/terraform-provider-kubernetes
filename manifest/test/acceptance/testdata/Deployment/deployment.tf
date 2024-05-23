# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata = {
      name      = var.name
      namespace = var.namespace
      labels = {
        app = "nginx"
      }
    }
    spec = {
      replicas = 2
      selector = {
        matchLabels = {
          app = "nginx"
        }
      }
      template = {
        metadata = {
          labels = {
            app = "nginx"
          }
        }
        spec = {
          containers = [
            {
              image = "nginx:1"
              name  = "nginx"
              ports = [
                {
                  containerPort = 80
                  protocol      = "TCP"
                },
              ]
            },
          ]
        }
      }
    }
  }
}
