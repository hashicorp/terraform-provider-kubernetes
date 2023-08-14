---
subcategory: "storage/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_csi_driver_v1"
description: |-
  The Container Storage Interface (CSI) is a standard for exposing arbitrary block and file storage systems to containerized workloads on Container Orchestration Systems (COs) like Kubernetes.
---

# kubernetes_csi_driver_v1

The [Container Storage Interface](https://kubernetes-csi.github.io/docs/introduction.html)
(CSI) is a standard for exposing arbitrary block and file storage systems to containerized workloads on Container 
Orchestration Systems (COs) like Kubernetes.

## Example Usage

```hcl
resource "kubernetes_csi_driver_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  spec {
    attach_required        = true
    pod_info_on_mount      = true
    volume_lifecycle_modes = ["Ephemeral"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard CSI driver's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) The Specification of the CSI Driver. 

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the csi driver that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the csi driver. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

### `spec`

#### Arguments

* `attach_required` - (Required) Indicates if the CSI volume driver requires an attachment operation.
* `pod_info_on_mount` - (Optional) Indicates that the CSI volume driver requires additional pod information (like podName, podUID, etc.) during mount operations.
* `volume_lifecycle_modes` - (Optional) A list of volume types the CSI volume driver supports. values can be `Persistent` and `Ephemeral`.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this csi driver that can be used by clients to determine when csi driver has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this csi driver. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Import

kubernetes_csi_driver_v1 can be imported using its name, e.g.

```
$ terraform import kubernetes_csi_driver_v1.example terraform-example
```
