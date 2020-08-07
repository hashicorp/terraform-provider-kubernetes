---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_all_namespaces"
sidebar_current: "docs-kubernetes-data-source-all-namespaces"
description: |-
  Lists all namespaces within a cluster.
---

# kubernetes_all_namespaces

This data source provides a mechanism for listing the names of all available namespaces in a Kubernetes cluster.
It can be used to check for existence of a specific namespaces or to apply another resource to all or a subset of existing namespaces in a cluster.

In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster resources between multiple users.

## Example Usage

```hcl
data "kubernetes_all_namespaces" "allns" {
  metadata {
    labels = {
      monitoring_enabled : true
    }
}

output "all-ns" {
  value = data.kubernetes_all_namespaces.allns.namespaces
}

output "ns-present" {
  value = contains(data.kubernetes_all_namespaces.allns.namespaces, "kube-system")
}

```

## Argument Reference

The following arguments are supported:

* `metadata` - (Optional) Standard service account's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) namespaces. May match selectors of replication controllers and services.
