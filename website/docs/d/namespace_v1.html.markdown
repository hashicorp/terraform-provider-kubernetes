---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_namespace_v1"
description: |-
  Queries attributes of a Namespace within the cluster.
---

# kubernetes_namespace_v1

This data source provides a mechanism to query attributes of any specific namespace within a Kubernetes cluster.
In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster resources between multiple users.

## Example Usage

```hcl
data "kubernetes_namespace_v1" "example" {
  metadata {
    name = "kube-system"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard object metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the namespace, must be unique. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

#### Attributes

* `annotations` - (Optional) An unstructured key value map stored with the namespace that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generation` - A sequence number representing a specific generation of the desired state.
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) namespaces. May match selectors of replication controllers and services.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `resource_version` - An opaque value that represents the internal version of this namespace that can be used by clients to determine when namespaces have changed. Read more about [concurrency control and consistency](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency).
* `uid` - The unique in time and space value for this namespace. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Attributes

* `finalizers` - An opaque list of values that must be empty to permanently remove object from storage. For more info: https://kubernetes.io/docs/tasks/administer-cluster/namespaces/

## Attribute Reference

* `wait_for_default_service_account` - (Optional) When set to `true` Terraform will wait until the default service account has been asynchronously created by Kubernetes when creating the namespace resource. This has the equivalent effect of creating a `resource_kubernetes_default_service_account` resource for dependent resources but allows a user to consume the "default" service account directly. The default behaviour (`false`) does not wait for the default service account to exist.

## Import

Namespaces can be imported using their name, e.g.

```
$ terraform import kubernetes_namespace_v1.n terraform-example-namespace
```
