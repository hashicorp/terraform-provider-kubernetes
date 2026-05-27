---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_nodes"
description: |-
  Gets nodes within a cluster.
---

# <no value>

<no value>

<no value> 

## Example usage

### All nodes

```terraform
data "kubernetes_nodes" "example" {}

output "node-ids" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.spec.0.provider_id]
}
```

### By label

```terraform
data "kubernetes_nodes" "example" {
  metadata {
    labels = {
      "kubernetes.io/os" = "linux"
    }
  }
}

output "linux-node-names" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.metadata.0.name]
}
```
