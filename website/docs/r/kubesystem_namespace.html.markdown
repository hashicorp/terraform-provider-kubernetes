---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_kubesystem_namespace"
sidebar_current: "docs-kubernetes-resource-kubesystem-namespace"
description: |-
  Manage the system generated kube-system Kubernetes namespace.
---

# kubernetes_kubesystem_namespace

Provides a resource to manage the system generated kube-system namespace.
Read more about namespaces at [Kubernetes reference](https://kubernetes.io/docs/user-guide/namespaces)

**This is an advanced resource**, and has special caveats to be aware of when
using it. Please read this document in its entirety before using this resource.

The `kubernetes_kubesystem_namespace` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead "adopts" it
into management.

## Example Usage

Add an annotation to the kube-system namespace:

```hcl
resource "kubernetes_kubesystem_namespace" "kube-system" {
  metadata {
    name = "kube-system"

    annotations {
      foo = "bar"
    }
  }
}

```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard namespace's [metadata](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata).

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the namespace that may be used to store arbitrary metadata. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more about [name idempotency](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#idempotency).
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) namespaces. May match selectors of replication controllers and services. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the namespace, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this namespace that can be used by clients to determine when namespaces have changed. Read more about [concurrency control and consistency](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency).
* `self_link` - A URL representing this namespace.
* `uid` - The unique in time and space value for this namespace. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

## Import

Namespaces can be imported using their name, e.g.

```
$ terraform import kubernetes_namespace.kube-system kube-system
```
