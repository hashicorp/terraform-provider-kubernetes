---
subcategory: "autoscaling/v1"
page_title: "Kubernetes: kubernetes_horizontal_pod_autoscaler_v1"
description: |-
  Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.
---

# <no value> 

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_horizontal_pod_autoscaler_v1" "example" {
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

## Import

Horizontal Pod Autoscaler can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_horizontal_pod_autoscaler_v1.example default/terraform-example
```
