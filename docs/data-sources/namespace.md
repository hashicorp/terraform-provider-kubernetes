---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_namespace"
description: |-
  Queries attributes of a Namespace within the cluster.
---

# <no value>

<no value>

<no value> 

## Example Usage

```terraform
data "kubernetes_namespace" "example" {
  metadata {
    name = "kube-system"
  }
}
```
