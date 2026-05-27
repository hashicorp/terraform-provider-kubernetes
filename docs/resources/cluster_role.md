---
subcategory: "rbac/v1"
page_title: "Kubernetes: kubernetes_cluster_role"
description: |-
  A ClusterRole creates a role at the cluster level and in all namespaces.
---

# <no value>

A ClusterRole creates a role at the cluster level and in all namespaces.

<no value>

## Example Usage

```terraform
resource "kubernetes_cluster_role" "example" {
  metadata {
    name = "terraform-example"
  }

  rule {
    api_groups = [""]
    resources  = ["namespaces", "pods"]
    verbs      = ["get", "list", "watch"]
  }
}
```

## Aggregation Rule Example Usage

```terraform
resource "kubernetes_cluster_role" "example" {
  metadata {
    name = "terraform-example"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        foo = "bar"
      }

      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }
}
```

## Import

ClusterRole can be imported using the name, e.g.

```
$ terraform import kubernetes_cluster_role.example terraform-name
```
