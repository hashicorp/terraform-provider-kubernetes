---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_node_taint"
description: |-
  A Node Taint is an anti-affinity rule allowing a Kubernetes node to repel a set of pods.
---

# kubernetes_node_taint

[Node affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) is a property of Pods that attracts them to a set of [nodes](https://kubernetes.io/docs/concepts/architecture/nodes/) (either as a preference or a hard requirement). Taints are the opposite -- they allow a node to repel a set of pods.

## Example Usage

```hcl
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


## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Metadata describing which Kubernetes node to apply the taint to.
* `field_manager` - (Optional) Set the name of the field manager for the node taint.
* `force` - (Optional) Force overwriting annotations that were created or edited outside of Terraform.
* `taint` - (Required) The taint configuration to apply to the node. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/).

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) The name of the node to apply the taint to

### `taint`

#### Arguments

* `key` - (Required, Forces new resource) The key of this node taint.
* `value` - (Required) The value of this node taint. Can be empty string.
* `effect` - (Required, Forces new resource) The scheduling effect to apply with this taint.  Must be one of: `NoSchedule`, `PreferNoSchedule`, `NoExecute`.

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it.
