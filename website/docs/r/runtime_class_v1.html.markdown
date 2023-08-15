---
layout: "kubernetes"
subcategory: "node/v1"
page_title: "Kubernetes: kubernetes_runtime_class_v1"
description: |-
  A runtime class is used to determine which container runtime is used to run all containers in a pod. 
---

# kubernetes_runtime_class_v1

A runtime class is used to determine which container runtime is used to run all containers in a pod.


## Example usage

```hcl
resource "kubernetes_runtime_class_v1" "example" {
  metadata {
    name = "myclass"
  }
  handler = "abcdeagh"
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard role's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `handler` - (Required) Specifies the underlying runtime and configuration that the CRI implementation will use to handle pods of this class
[Kubernetes reference](https://kubernetes.io/docs/concepts/containers/runtime-class/)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the role that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](hhttps://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the role. **Must match `selector`**.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the role, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this role that can be used by clients to determine when role has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this role. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Import

Runtime class can be imported using the name only, e.g.

```
$ terraform import kubernetes_runtime_class_v1.example myclass
```


