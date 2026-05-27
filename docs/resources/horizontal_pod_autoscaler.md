---
subcategory: "autoscaling/v1"
page_title: "Kubernetes: kubernetes_horizontal_pod_autoscaler"
description: |-
  Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_horizontal_pod_autoscaler" "example" {
  metadata {
    name = "terraform-example"
  }

  spec {
    max_replicas = 10
    min_replicas = 8

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }
  }
}
```

## Example Usage, with `metric`

```terraform
resource "kubernetes_horizontal_pod_autoscaler" "example" {
  metadata {
    name = "test"
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }

    metric {
      type = "External"
      external {
        metric {
          name = "latency"
          selector {
            match_labels = {
              lb_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "100"
        }
      }
    }
  }
}
```

## Example Usage, with `behavior`

```terraform
resource "kubernetes_horizontal_pod_autoscaler" "example" {
  metadata {
    name = "test"
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 300
        select_policy                = "Min"
        policy {
          period_seconds = 120
          type           = "Pods"
          value          = 1
        }

        policy {
          period_seconds = 310
          type           = "Percent"
          value          = 100
        }
      }
      scale_up {
        stabilization_window_seconds = 600
        select_policy                = "Max"
        policy {
          period_seconds = 180
          type           = "Percent"
          value          = 100
        }
        policy {
          period_seconds = 600
          type           = "Pods"
          value          = 5
        }
      }
    }
  }
}
```

## Support for multiple and custom metrics

The provider currently supports two version of the HorizontalPodAutoscaler API resource.

If you wish to use `autoscaling/v1` use the `target_cpu_utilization_percentage` field.

If you wish to use `autoscaling/v2beta2` then set one or more `metric` fields.

## Import

Horizontal Pod Autoscaler can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_horizontal_pod_autoscaler.example default/terraform-example
```
