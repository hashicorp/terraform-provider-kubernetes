---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_storage_class"
sidebar_current: "docs-kubernetes-resource-storage-class"
description: |-
  Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.
---

# kubernetes_storage_class

Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform.

Read more at http://blog.kubernetes.io/2017/03/dynamic-provisioning-and-storage-classes-kubernetes.html

## Example Usage

```hcl
resource "kubernetes_storage_class" "example" {
  metadata {
    name = "terraform-example"
  }
  storage_provisioner = "kubernetes.io/gce-pd"
  parameters {
  	type = "pd-standard"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard storage class's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* `parameters` - (Optional) The parameters for the provisioner that should create volumes of this storage class.
	Read more about [available parameters](https://kubernetes.io/docs/concepts/storage/storage-classes/#parameters).
* `storage_provisioner` - (Required) Indicates the type of the provisioner

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the storage class that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the storage class. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the storage class, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this storage class that can be used by clients to determine when storage class has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `self_link` - A URL representing this storage class.
* `uid` - The unique in time and space value for this storage class. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

## Import

kubernetes_storage_class can be imported using its name, e.g.

```
$ terraform import kubernetes_storage_class.example terraform-example
```
