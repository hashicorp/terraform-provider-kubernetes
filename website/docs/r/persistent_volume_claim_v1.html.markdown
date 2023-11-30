---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_persistent_volume_claim_v1"
description: |-
  This resource allows the user to request for and claim to a persistent volume.
---

# kubernetes_persistent_volume_claim_v1

This resource allows the user to request for and claim to a persistent volume.

## Example Usage

```hcl
resource "kubernetes_persistent_volume_claim_v1" "example" {
  metadata {
    name = "exampleclaimname"
  }
  spec {
    access_modes = ["ReadWriteMany"]
    resources {
      requests = {
        storage = "5Gi"
      }
    }
    volume_name = "${kubernetes_persistent_volume_v1.example.metadata.0.name}"
  }
}

resource "kubernetes_persistent_volume_v1" "example" {
  metadata {
    name = "examplevolumename"
  }
  spec {
    capacity = {
      storage = "10Gi"
    }
    access_modes = ["ReadWriteMany"]
    persistent_volume_source {
      gce_persistent_disk {
        pd_name = "test-123"
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard persistent volume claim's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the desired characteristics of a volume requested by a pod author. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
* `wait_until_bound` - (Optional) Whether to wait for the claim to reach `Bound` state (to find volume in which to claim the space)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the persistent volume claim that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the persistent volume claim. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the persistent volume claim, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the persistent volume claim must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this persistent volume claim that can be used by clients to determine when persistent volume claim has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this persistent volume claim. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `access_modes` - (Required) A set of the desired access modes the volume should have. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes)
* `resources` - (Required) A list of the minimum resources the volume should have. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources)
* `selector` - (Optional) A label query over volumes to consider for binding.
* `volume_name` - (Optional) The binding reference to the PersistentVolume backing this claim.
* `storage_class_name` - (Optional) Name of the storage class requested by the claim.
* `volume_mode` - (Optional) Defines what type of volume is required by the claim. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#volume-mode)

### `match_expressions`

#### Arguments

* `key` - (Optional) The label key that the selector applies to.
* `operator` - (Optional) A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.
* `values` - (Optional) An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.


### `resources`

#### Arguments

* `limits` - (Optional) Map describing the maximum amount of compute resources allowed. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
* `requests` - (Optional) Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

### `selector`

#### Arguments

* `match_expressions` - (Optional) A list of label selector requirements. The requirements are ANDed.
* `match_labels` - (Optional) A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.

## Import

Persistent Volume Claim can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_persistent_volume_claim_v1.example default/example-name
```
