---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_all_namespaces"
description: |-
  Lists all namespaces within a cluster.
---

# kubernetes_all_namespaces

This data source provides a mechanism for listing the names of all available namespaces in a Kubernetes cluster.
It can be used to check for existence of a specific namespaces or to apply another resource to all or a subset of existing namespaces in a cluster.

In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster resources between multiple users.

## Example Usage

```hcl
data "kubernetes_all_namespaces" "allns" {}

output "all-ns" {
  value = data.kubernetes_all_namespaces.allns.namespaces
}

output "ns-present" {
  value = contains(data.kubernetes_all_namespaces.allns.namespaces, "kube-system")
}

```
