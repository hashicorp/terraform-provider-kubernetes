---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service_account"
description: |-
  A service account provides an identity for processes that run in a Pod.
---

# kubernetes_service_account

A service account provides an identity for processes that run in a Pod.  This data source reads the service account and makes specific attributes available to Terraform.

Read more at [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/)

## Example Usage

```hcl
data "kubernetes_service_account" "example" {
  metadata {
    name = "terraform-example"
  }
}

data "kubernetes_secret" "example" {
  metadata {
    name = "${data.kubernetes_service_account.example.default_secret_name}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service account's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the service account, must be unique. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the service account must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service account that can be used by clients to determine when service account has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this service account. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Attribute Reference

* `image_pull_secret` - A list of image pull secrets associated with the service account.
* `secret` - A list of secrets associated with the service account.
* `default_secret_name` - (Deprecated) Name of the default secret, containing service account token, created & managed by the service. By default, the provider will try to find the secret containing the service account token that Kubernetes automatically created for the service account. Where there are multiple tokens and the provider cannot determine which was created by Kubernetes, this attribute will be empty. When only one token is associated with the service account, the provider will return this single token secret.

  Starting from version `1.24.0` by default Kubernetes does not automatically generate tokens for service accounts. That leads to the situation when `default_secret_name` cannot be computed and thus will be an empty string. In order to create a service account token, please [use `kubernetes_secret_v1` resource](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/secret_v1#example-usage-service-account-token)

### `image_pull_secret`

#### Attributes

* `name` - Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `secret`

#### Attributes

* `name` - Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
