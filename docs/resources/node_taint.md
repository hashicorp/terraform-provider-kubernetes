---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_node_taint"
description: |-
  A Node Taint is an anti-affinity rule allowing a Kubernetes node to repel a set of pods.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_node_taint" "example" {
  metadata {
    name = "my-node.my-cluster.k8s.local"
  }
  taint {
    key    = "node-role.kubernetes.io/example"
    value  = "true"
    effect = "NoSchedule"
  }
}
```

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it.
