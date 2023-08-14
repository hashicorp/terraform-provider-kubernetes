---
subcategory: "storage/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_storage_class_v1"
description: |-
  Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.
---

# kubernetes_storage_class_v1

Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.

Read more at https://kubernetes.io/blog/2017/03/dynamic-provisioning-and-storage-classes-kubernetes/

## Example Usage

```hcl
resource "kubernetes_storage_class_v1" "example" {
  metadata {
    name = "terraform-example"
  }
  storage_provisioner = "kubernetes.io/gce-pd"
  reclaim_policy      = "Retain"
  parameters = {
    type = "pd-standard"
  }
  mount_options = ["file_mode=0700", "dir_mode=0777", "mfsymlinks", "uid=1000", "gid=1000", "nobrl", "cache=none"]
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard storage class's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `parameters` - (Optional) The parameters for the provisioner that should create volumes of this storage class.
	Read more about [available parameters](https://kubernetes.io/docs/concepts/storage/storage-classes/#parameters).
* `storage_provisioner` - (Required) Indicates the type of the provisioner
* `reclaim_policy` - (Optional) Indicates the reclaim policy to use.  If no reclaimPolicy is specified when a StorageClass object is created, it will default to Delete.
* `volume_binding_mode` - (Optional) Indicates when volume binding and dynamic provisioning should occur.
* `allow_volume_expansion` - (Optional) Indicates whether the storage class allow volume expand, default true.
* `mount_options` - (Optional) Persistent Volumes that are dynamically created by a storage class will have the mount options specified.
* `allowed_topologies` - (Optional) Restrict the node topologies where volumes can be dynamically provisioned. See [allowed_topologies](#allowed_topologies)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the storage class that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the storage class. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the storage class, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `allowed_topologies`

#### Arguments

* `match_label_expressions` - (Optional) A list of topology selector requirements by labels. See [match_label_expressions](#match_label_expressions)

### `match_label_expressions`

#### Arguments

* `key` - (Optional) The label key that the selector applies to.
* `values` - (Optional) An array of string values. One value must match the label to be selected.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this storage class that can be used by clients to determine when storage class has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this storage class. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Import

kubernetes_storage_class_v1 can be imported using its name, e.g.

```
$ terraform import kubernetes_storage_class_v1.example terraform-example
```
