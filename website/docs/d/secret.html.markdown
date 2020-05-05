---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_secret"
sidebar_current: "docs-kubernetes-resource-secret"
description: |-
  The resource provides mechanisms to inject containers with sensitive information while keeping containers agnostic of Kubernetes.
---

# kubernetes_secret

The resource provides mechanisms to inject containers with sensitive information, such as passwords, while keeping containers agnostic of Kubernetes.
Secrets can be used to store sensitive information either as individual properties or coarse-grained entries like entire files or JSON blobs.
The resource will by default create a secret which is available to any pod in the specified (or default) namespace.

~> Read more about security properties and risks involved with using Kubernetes secrets: [Kubernetes reference](https://kubernetes.io/docs/user-guide/secrets/#security-properties)

~> **Note:** All arguments including the secret data will be stored in the raw state as plain-text. [Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
data "kubernetes_secret" "example" {
  metadata {
    name = "basic-auth"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard secret's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the secret, must be unique. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the secret must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this secret that can be used by clients to determine when secret has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this secret.
* `uid` - The unique in time and space value for this secret. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

## Attribute Reference

* `data` - A map of the secret data.
* `type` - The secret type. Defaults to `Opaque`. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/c7151dd8dd7e487e96e5ce34c6a416bb3b037609/contributors/design-proposals/auth/secrets.md#proposed-design)
