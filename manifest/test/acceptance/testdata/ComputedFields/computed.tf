# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
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

resource "kubernetes_manifest" "deployment_resource_diff" {
computed_fields = ["spec.template.spec.containers[0].resources.limits"]
    manifest = {
        apiVersion = "apps/v1"
        kind       = "Deployment"

        metadata = {
            name = var.name
            namespace = var.namespace
        }

        spec = {
    replicas = 3

    selector = {
      matchLabels = {
        test = "MyExampleApp"
      }
    }

    template = {
      metadata= {
        labels = {
          test = "MyExampleApp"
        }
      }
      

      spec = {
        containers = [{
          image = "nginx:1.21.6"
          name  = "example"

          resources = {
            limits = {
              cpu    = "0.25"
              memory = "512Mi"
            }
            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }]
      }
    }
  }
  }
}