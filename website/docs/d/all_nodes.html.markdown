---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_all_nodes"
description: |-
  Lists all nodes within a cluster.
---

# kubernetes_all_nodes

This data source provides a mechanism for listing the names of all nodes in a
kubernetes cluster.

It can be used to check for the existance of a specific node or to lookup a node
to apply a taint with the `kubernetes_node_taint` resource.

## Example usage

```hcl
data "kubernetes_all_nodes" "example" {}

output "all-nodes" {
   value = data.kubernetes_all_nodes.example.nodes
}
```
