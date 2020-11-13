---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service_account"
description: |-
  A service account provides an identity for processes that run in a Pod.
---

# kubernetes_service_account

A service account provides an identity for processes that run in a Pod.

Read more at [Kubernetes reference](https://kubernetes.io/docs/admin/service-accounts-admin/)

## Example Usage

```hcl
resource "kubernetes_service_account" "example" {
  metadata {
    name = "terraform-example"
  }
  secret {
    name = "${kubernetes_secret.example.metadata.0.name}"
  }
}

resource "kubernetes_secret" "example" {
  metadata {
    name = "terraform-example"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service account's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `image_pull_secret` - (Optional) A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret)
* `secret` - (Optional) A list of secrets allowed to be used by pods running using this Service Account. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/secrets)
* `automount_service_account_token` - (Optional) Boolean, `true` to enable automatic mounting of the service account token. Defaults to `true`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the service account that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service account. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)

* `name` - (Optional) Name of the service account, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the service account must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service account that can be used by clients to determine when service account has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this service account.
* `uid` - The unique in time and space value for this service account. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `image_pull_secret`

#### Arguments

* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### `secret`

#### Arguments

* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `default_secret_name` - Name of the default secret, containing service account token, created & managed by the service.

## Import

Service account can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_service_account.example default/terraform-example
```
