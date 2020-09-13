---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_all_nodes"
description: |-
  Lists all nodes within a cluster.
---

# kubernetes_all_nodes

This data source provides a mechanism for listing the names of all available nodes in a Kubernetes cluster.
It can be used to check for existence of a specific node or to apply another resource to all or a subset of existing nodes in a cluster.

## Example Usage

```hcl
data "kubernetes_all_nodes" "allnodes" {}

output "all-nodes" {
  value = data.kubernetes_all_nodes.allnodes.nodes
}

output "node-present" {
  value = contains(data.kubernetes_all_nodes.allnodes.nodes, "kube-system")
}

```
