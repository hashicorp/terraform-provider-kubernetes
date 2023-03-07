# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_svc" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Service"
    "metadata" = {
      "labels" = {
        "app" = "nginx"
      }
      "name"      = var.name
      "namespace" = var.namespace
    }
    "spec" = {
      "clusterIP" = "None"
      "ports" = [
        {
          "name"     = "web"
          "port"     = 80
          "protocol" = "TCP"
        },
      ]
      "selector" = {
        "app" = "nginx"
      }
    }
  }
}
resource "kubernetes_manifest" "test" {
  manifest = {
    "apiVersion" = "apps/v1"
    "kind"       = "StatefulSet"
    "metadata" = {
      "name"      = var.name
      "namespace" = var.namespace
    }
    "spec" = {
      "replicas" = 2
      "selector" = {
        "matchLabels" = {
          "app" = "nginx"
        }
      }
      "serviceName" = var.name
      "template" = {
        "metadata" = {
          "labels" = {
            "app" = "nginx"
          }
        }
        "spec" = {
          "containers" = [
            {
              "image" = "nginx:1"
              "name"  = "nginx"
              "ports" = [
                {
                  "containerPort" = 80
                  "name"          = "web"
                  "protocol"      = "TCP"
                },
              ]
              "volumeMounts" = [
                {
                  "mountPath" = "/usr/share/nginx/html"
                  "name"      = "www"
                },
              ]
            },
          ]
        }
      }
      "volumeClaimTemplates" = [
        {
          "metadata" = {
            "name" = "www"
          }
          "spec" = {
            "accessModes" = [
              "ReadWriteOnce",
            ]
            "resources" = {
              "requests" = {
                "storage" = "1Gi"
              }
            }
          }
        },
      ]
    }
  }
}
