# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource kubernetes_manifest wait_for_rollout {
  manifest = {
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata = {
      name       = var.name
      namespace  = var.namespace
    }
    spec = {
      replicas = 2
      selector = {
        matchLabels = {
          app = "tf-acc-test"
        }
      }
      template = {
        metadata = {
          labels = {
            app = "tf-acc-test"
          }
        }
        spec = {
          containers = [
            {
              image           = "nginx:1.19.4"
              imagePullPolicy = "IfNotPresent"
              name            = "tf-acc-test"
              readinessProbe  = {
                httpGet = {
                  port = 80
                  path = "/"
                }
                initialDelaySeconds = 10
              }
            },
          ]
        }
      }
    }
  }

  wait {
    rollout = true
  }
}