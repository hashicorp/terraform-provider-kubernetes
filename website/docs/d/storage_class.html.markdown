---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_storage_class"
sidebar_current: "docs-kubernetes-data-source-storage-class"
description: |-
  Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.
---

# kubernetes_storage_class

Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.

Read more at https://kubernetes.io/blog/2017/03/dynamic-provisioning-and-storage-classes-kubernetes/

## Example Usage

```
data "kubernetes_storage_class" "example" {
  metadata {
    name = "terraform-example"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard storage class's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)


## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the storage class, must be unique. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this storage class that can be used by clients to determine when storage class has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this storage class.
* `uid` - The unique in time and space value for this storage class. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)


## Argument Reference

The following attributes are exported:

* `parameters` - The parameters for the provisioner that creates volume of this storage class.
	Read more about [available parameters](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#parameters).
* `storage_provisioner` - Indicates the type of the provisioner this storage class represents
* `reclaim_policy` - Indicates the reclaim policy used.
* `volume_binding_mode` - Indicates when volume binding and dynamic provisioning should occur.
* `allow_volume_expansion` - Indicates whether the storage class allow volume expand.
