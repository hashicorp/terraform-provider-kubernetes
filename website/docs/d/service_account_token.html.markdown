---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service_account_token"
sidebar_current: "docs-kubernetes-data-source-service-account-token"
description: |-
  Use to read service account token secrets.
---

# kubernetes_service_account_token

Use this data source to read service account token secrets.

While `kubernetes_secret` data resource can read the service account token secrets, this tends to run into problems such as
being unable to access the `ca.crt` value.  

## Example Usage

```hcl
data "kubernetes_service_account_token" "test" {
  metadata {
    name = "example-service-bemorq6tot"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard secret's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)

## Attributes

* `data` - Data defines the attributes for this service account token.

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the secret, must be unique. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the secret must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this secret that can be used by clients to determine when secret has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this secret.
* `uid` - The unique in time and space value for this secret. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `data`

#### Attributes

* `ca_crt` - CA certificate for the apiserver - this is 'ca.crt' in the underlying secret
* `namespace` - Namespace that the service account token is application for.
* `token` -  Bearer token used to authenticate against the apiserver.
