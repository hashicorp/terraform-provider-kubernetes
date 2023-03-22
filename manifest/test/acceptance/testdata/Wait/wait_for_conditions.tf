# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "v1"
    kind       = "Pod"

    metadata = {
      name      = var.name
      namespace = var.namespace

      annotations = {
        "test.terraform.io" = "test"
      }

      labels = {
        app = "nginx"
      }
    }

    spec = {
      containers = [
        {
          name  = "nginx"
          image = "nginx:1.19"

          readinessProbe = {
            initialDelaySeconds = 10

            httpGet = {
              path = "/"
              port = 80
            }
          }
        }
      ]
    }
  }

  wait {
    condition {
      type = "Ready"
      status = "True"
    }

    condition {
      type = "ContainersReady"
      status = "True"
    }
  }
}
