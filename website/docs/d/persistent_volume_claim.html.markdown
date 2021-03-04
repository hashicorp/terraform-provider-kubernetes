---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_persistent_volume_claim"
description: |-
  Queries attributes of a PersistentVolumeClaim (PVC).
---

# kubernetes_persistent_volume_claim

A PersistentVolumeClaim (PVC) is a request for storage by a user. This data source retrieves information about the specified PVC.


## Example Usage

```hcl
data "kubernetes_persistent_volume_claim" "example" {
  metadata {
    name = "terraform-example"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard persistent volume claim's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)



## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Optional) Name of the persistent volume claim, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the persistent volume claim must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this persistent volume claim that can be used by clients to determine when persistent volume claim has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this persistent volume claim. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `spec`

#### Attributes

* `access_modes` - A set of the desired access modes the volume should have. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes-1)
* `selector` - Claims can specify a label selector to further filter the set of volumes. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/persistent-volumes#selector)
* `volume_name` - The binding reference to the PersistentVolume backing this claim.
* `storage_class_name` - Name of the storage class requested by the claim.

## Import

Persistent Volume Claim can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_persistent_volume_claim.example default/example-name
```
