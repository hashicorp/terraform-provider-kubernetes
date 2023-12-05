---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_nodes"
description: |-
  Gets nodes within a cluster.
---

# kubernetes_nodes

This data source provides a mechanism for listing the names of nodes in a kubernetes cluster.

By default, all nodes in the cluster are returned, but queries by node label are also supported.

It can be used to check for the existence of a specific node or to lookup a node to apply a taint with the `kubernetes_node_taint` resource.

## Example usage

### All nodes

```hcl
data "kubernetes_nodes" "example" {}

output "node-ids" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.spec.0.provider_id]
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

output "linux-node-names" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.metadata.0.name]
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - Metadata describing which nodes to return.

### Attributes

* `metadata` - Metadata describing the node. See [metadata](#metadata) for more
  info.
* `spec` - Defines the behavior of the node. See [spec](#spec) for more info.
* `status` - Status information for the node.  See [status](#status) for more
  info.

## Nested Blocks

### `metadata`

#### Arguments

* `labels` - (Required) Map of string keys and values that can be used to narrow the selection of nodes returned.

#### Attributes

* `name` - Name of the node, must be unique. 
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the node.
* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this node that can be used by clients to determine when the node has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this node . For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Attributes

* `pod_cidr` - (Optional) The pod IP range assigned to the node
* `pod_cidrs` - (Optional) A list of IP address ranges assigned to the node for
  usage by pods on that node.
* `provider_id` - (Optional) ID of the node assigned by the cloud provider.
* `unschedulable` - Controls the schedulability of new pods on the node.  By default, node is schedulable.
* `taints` - (Optional) Taints applied to the node.  See [taints](#taints) for
  more info.

### `status`

### Attributes

* `addresses` - (Optional) A set of IP address(es) and/or Hostname assigned to the node. See [addresses](#addresses) and [Kubernetes reference](https://kubernetes.io/docs/concepts/architecture/nodes/#addresses/node/#info) for more info.
* `allocatable` - (Optional) The total resources of a node.
* `capacity` - (Optional) The resources of a node that are available for scheduling.
* `node_info` - (Optional) A set of ids/uuids to uniquely identify the node. See [node_info](#node_info) for more info. [Kubernetes reference](https://kubernetes.io/docs/concepts/nodes/node/#info)

### `addresses`

#### Attributes

* `type` - Type of the address: HostName, ExternalIP or InternalIP.
* `address` - The IP (if type is ExternalIP or InternalIP) or the hostname (if type is HostName).

### `node_info`

#### Attributes

* `machine_id` - Machine ID reported by the node see [main(5)
  machine-id](http://man7.org/linux/man-pages/man5/machine-id.5.html) for more info.
* `system_uuid` - System UUID reported by the node. This field is
  specific to [Red Hat hosts](https://access.redhat.com/documentation/en-us/red_hat_subscription_management/1/html/rhsm/uuid)
* `boot_id` - Boot ID reported by the node.
* `kernel_version` - Kernel Version reported by the node from `uname -r`
* `os_image` - OS Image reported by the node from `/etc/os-release`
* `container_runtime_version` ContainerRuntime Version reported by the node through runtime remote API
* `kubelet_version` - Kubelet Version reported by the node.
* `kube_proxy_version` - KubeProxy Version reported by the node.
* `operating_system` - The Operating System reported by the node
* `architecture` - The Architecture reported by the node

### `taints`

#### Attributes

* `key` - The taint key to be applied to a node.
* `value` - (Optional) The taint value corresponding to the taint key.
* `effect` - The effect of the taint on pods that do not tolerate the taint. Valid effects are `NoSchedule`, `PreferNoSchedule` and `NoExecute`.
