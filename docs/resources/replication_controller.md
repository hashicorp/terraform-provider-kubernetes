---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_replication_controller"
description: |-
  A Replication Controller ensures that a specified number of pod “replicas” are running at any one time. In other words, a Replication Controller makes sure that a pod or homogeneous set of pods are always up and available. If there are too many pods, it will kill some. If there are too few, the Replication Controller will start more.
---

# <no value>

A Replication Controller ensures that a specified number of pod “replicas” are running at any one time. In other words, a Replication Controller makes sure that a pod or homogeneous set of pods are always up and available. If there are too many pods, it will kill some. If there are too few, the Replication Controller will start more.

~> **WARNING:** In many cases it is recommended to create a Deployment instead of a Replication Controller.

<no value>

## Example Usage

```terraform
resource "kubernetes_replication_controller" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      test = "MyExampleApp"
    }
  }

  spec {
    selector = {
      test = "MyExampleApp"
    }
    template {
      metadata {
        labels = {
          test = "MyExampleApp"
        }
        annotations = {
          "key1" = "value1"
        }
      }

      spec {
        container {
          image = "nginx:1.21.6"
          name  = "example"

          liveness_probe {
            http_get {
              path = "/"
              port = 80

              http_header {
                name  = "X-Custom-Header"
                value = "Awesome"
              }
            }

            initial_delay_seconds = 3
            period_seconds        = 3
          }

          resources {
            limits = {
              cpu    = "0.5"
              memory = "512Mi"
            }
            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
      }
    }
  }
}
```

## Import

Replication Controller can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_replication_controller.example default/terraform-example
```

~> **NOTE:** Imported `kubernetes_replication_controller` resource will only have their fields from the `spec.template.spec` block in the state. Deprecated fields at the `spec.template` level are not updated during import. Configurations using the deprecated fields should be updated to only use the new fields under `spec.template.spec`.
