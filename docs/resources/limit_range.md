---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_limit_range"
description: |-
  Limit Range sets resource usage limits (e.g. memory, cpu, storage) for supported kinds of resources in a namespace.
---

# <no value>

Limit Range sets resource usage limits (e.g. memory, cpu, storage) for supported kinds of resources in a namespace.

Read more in [the official docs](https://kubernetes.io/docs/concepts/policy/limit-range/).

<no value>

## Example Usage

```terraform
resource "kubernetes_limit_range" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    limit {
      type = "Pod"
      max = {
        cpu    = "200m"
        memory = "1024Mi"
      }
    }
    limit {
      type = "PersistentVolumeClaim"
      min = {
        storage = "24M"
      }
    }
    limit {
      type = "Container"
      default = {
        cpu    = "50m"
        memory = "24Mi"
      }
    }
  }
}
```

## Import

Limit Range can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_limit_range.example default/terraform-example
```
