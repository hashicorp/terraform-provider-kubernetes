---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_priority_class"
sidebar_current: "docs-kubernetes-resource-priority-class"
description: |-
  A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.
---

# kubernetes_priority_class

A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.

## Example Usage

```hcl
resource "kubernetes_priority_class" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard resource quota's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `value` - (Required, Forces new resource) The value of this priority class. This is the actual priority that pods receive when they have the name of this class in their pod spec.
* `description` - (Optional) An arbitrary string that usually provides guidelines on when this priority class should be used.
* `global_default` - (Optional) Boolean that specifies whether this PriorityClass should be considered as the default priority for pods that do not have any priority class.

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

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this resource quota that can be used by clients to determine when resource quota has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this resource quota.
* `uid` - The unique in time and space value for this resource quota. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

## Import

Priority Class can be imported using its name, e.g.

```
$ terraform import kubernetes_priority_class.example terraform-example
```
