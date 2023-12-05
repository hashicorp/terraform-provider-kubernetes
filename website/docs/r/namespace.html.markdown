---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_namespace"
description: |-
  Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces.
---

# kubernetes_namespace

Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces.
Read more about namespaces at [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

## Example Usage

```hcl
resource "kubernetes_namespace" "example" {
  metadata {
    annotations = {
      name = "example-annotation"
    }

    labels = {
      mylabel = "label-value"
    }

    name = "terraform-example-namespace"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard namespace's [metadata](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata).

### Timeouts

`kubernetes_namespace` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `delete` - Default `5 minutes`

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the namespace that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/).

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more about [name idempotency](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency).
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) namespaces. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/).

* `name` - (Optional) Name of the namespace, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names).

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this namespace that can be used by clients to determine when namespaces have changed. Read more about [concurrency control and consistency](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency).
* `uid` - The unique in time and space value for this namespace. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids).

## Attribute Reference

* `wait_for_default_service_account` - (Optional) When set to `true` Terraform will wait until the default service account has been asynchronously created by Kubernetes when creating the namespace resource. This has the equivalent effect of creating a `kubernetes_default_service_account` resource for dependent resources but allows a user to consume the "default" service account directly. The default behaviour (`false`) does not wait for the default service account to exist.

## Import

Namespaces can be imported using their name, e.g.

```
$ terraform import kubernetes_namespace.n terraform-example-namespace
```

