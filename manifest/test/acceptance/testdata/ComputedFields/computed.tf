# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_config_map" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "annotations" = {
        "tf-k8s-acc" = "true"
      }
      "name"      = var.name
      "namespace" = var.namespace
    }
  }
}

resource "kubernetes_manifest" "test_pod" {
  computed_fields =  ["spec.containers[0].resources.limits[\"cpu\"]"]
  manifest = {
    apiVersion = "v1"
    kind       = "Pod"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      containers = [
        {
          name  = "my-container"
          image = "nginx:latest"
          resources = {
            limits = {
              memory = "1.2G"
              cpu    = "500m"
            }
            requests = {
              memory = "1.1G"
              cpu    = "250m"
            }
          }
        }
      ]
    }
  }
}
