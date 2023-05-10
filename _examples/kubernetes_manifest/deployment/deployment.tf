# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test-deployment" {

  manifest = {
    "apiVersion" = "apps/v1"
    "kind"       = "Deployment"
    "metadata" = {
      "labels" = {
        "app" = "nginx"
      }
      "name"      = "nginx-deployment"
      "namespace" = "default"
    }
    "spec" = {
      "replicas" = 3
      "selector" = {
        "matchLabels" = {
          "app" = "nginx"
        }
      }
      "template" = {
        "metadata" = {
          "labels" = {
            "app" = "nginx"
          }
        }
        "spec" = {
          "containers" = [
            {
              "image" = "nginx:1.14.2"
              "name"  = "nginx"
              "ports" = [
                {
                  "containerPort" = 80
                  "protocol"      = "TCP"
                },
              ]
              "volumeMounts" = [
                {
                  "mountPath" = "/foobar/"
                  "name"      = "vol-foobar"
                },
              ]
            },
          ],
          "volumes" = [
            {
              "emptyDir" = {}
              "name"     = "vol-foobar"
            },
          ]
        }
      }
    }
  }

}
