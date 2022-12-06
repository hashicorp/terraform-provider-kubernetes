---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_nodes"
description: |-
  Lists nodes within a cluster.
---

# kubernetes_nodes

This data source provides a mechanism for listing the names of nodes in a kubernetes cluster.

By default, all nodes in the cluster are returned, but queries by node label are also supported.

It can be used to check for the existance of a specific node or to lookup a node to apply a taint with the `kubernetes_node_taint` resource.

## Example usage

### All nodes
```hcl
data "kubernetes_nodes" "example" {}

output "all-nodes" {
  value = data.kubernetes_nodes.example.nodes
}
```

### By label
```hcl
data "kubernetes_nodes" "example" {
  metadata {
    labels = {
      "kubernetes.io/os" = "linux"
    }
  }
}

output "linux-nodes" {
  value = data.kubernetes_nodes.example.nodes
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - Metadata describing which nodes to return.

## Nested Blocks

### `metadata`

#### Arguments

* `labels` - (Required) Map of string keys and values that can be used to narrow the selection of nodes returned.
