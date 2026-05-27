---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_all_namespaces"
description: |-
  Lists all namespaces within a cluster.
---

# <no value>

<no value> 

<no value>

## Example Usage

```terraform
data "kubernetes_all_namespaces" "allns" {}

output "all-ns" {
  value = data.kubernetes_all_namespaces.allns.namespaces
}

output "ns-present" {
  value = contains(data.kubernetes_all_namespaces.allns.namespaces, "kube-system")
}
```
