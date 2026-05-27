---
subcategory: "apps/v1"
page_title: "Kubernetes: kubernetes_daemon_set_v1"
description: |-
  A DaemonSet ensures that all (or some) Nodes run a copy of a Pod. As nodes are added to the cluster, Pods are added to them. As nodes are removed from the cluster, those Pods are garbage collected. Deleting a DaemonSet will clean up the Pods it created.
---

# <no value>

<no value>

<no value> 

## Example Usage

```terraform
resource "kubernetes_daemon_set_v1" "example" {
  metadata {
    name      = "terraform-example"
    namespace = "something"
    labels = {
      test = "MyExampleApp"
    }
  }

  spec {
    selector {
      match_labels = {
        test = "MyExampleApp"
      }
    }

    template {
      metadata {
        labels = {
          test = "MyExampleApp"
        }
      }

      spec {
        container {
          image = "nginx:1.21.6"
          name  = "example"

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

        }
      }
    }
  }
}
```

## Import

DaemonSet can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_daemon_set_v1.example default/terraform-example
```
