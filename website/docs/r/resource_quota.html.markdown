---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_resource_quota"
sidebar_current: "docs-kubernetes-resource-resource-quota"
description: |-
  A resource quota provides constraints that limit aggregate resource consumption per namespace. It can limit the quantity of objects that can be created in a namespace by type, as well as the total amount of compute resources that may be consumed by resources in that project.
---

# kubernetes_resource_quota

A resource quota provides constraints that limit aggregate resource consumption per namespace. It can limit the quantity of objects that can be created in a namespace by type, as well as the total amount of compute resources that may be consumed by resources in that project.


## Example Usage

```hcl
resource "kubernetes_resource_quota" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    hard {
      pods = 10
    }
    scopes = ["BestEffort"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard resource quota's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)
* `spec` - (Optional) Spec defines the desired quota. [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the resource quota that may be used to store arbitrary metadata. 
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the resource quota. May match selectors of replication controllers and services. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the resource quota, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the resource quota must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this resource quota that can be used by clients to determine when resource quota has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this resource quota.
* `uid` - The unique in time and space value for this resource quota. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `spec`

#### Arguments

* `hard` - (Optional) The set of desired hard limits for each named resource. For more info see http://releases.k8s.io/HEAD/docs/design/admission_control_resource_quota.md#admissioncontrol-plugin-resourcequota
* `scopes` - (Optional) A collection of filters that must match each object tracked by a quota. If not specified, the quota matches all objects.

## Import

Resource Quota can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_resource_quota.example default/terraform-example
```