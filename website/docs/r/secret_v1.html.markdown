---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_secret_v1"
description: |-
  The resource provides mechanisms to inject containers with sensitive information while keeping containers agnostic of Kubernetes.
---

# kubernetes_secret_v1

The resource provides mechanisms to inject containers with sensitive information, such as passwords, while keeping containers agnostic of Kubernetes.
Secrets can be used to store sensitive information either as individual properties or coarse-grained entries like entire files or JSON blobs.
The resource will by default create a secret which is available to any pod in the specified (or default) namespace.

~> Read more about security properties and risks involved with using Kubernetes secrets: [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/secret/#information-security-for-secrets)

~> **Note:** All arguments including the secret data will be stored in the raw state as plain-text. [Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "basic-auth"
  }

  data = {
    username = "admin"
    password = "P4ssw0rd"
  }

  type = "kubernetes.io/basic-auth"
}
```

## Example Usage (Docker config)

### Docker config file

```hcl
resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "docker-cfg"
  }

  data = {
    ".dockerconfigjson" = "${file("${path.module}/.docker/config.json")}"
  }

  type = "kubernetes.io/dockerconfigjson"
}
```

### Username and password

```hcl
resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "docker-cfg"
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        "${var.registry_server}" = {
          "username" = var.registry_username
          "password" = var.registry_password
          "email"    = var.registry_email
          "auth"     = base64encode("${var.registry_username}:${var.registry_password}")
        }
      }
    })
  }
}
```

This is equivalent to the following kubectl command:

```sh
$ kubectl create secret docker-registry docker-cfg --docker-server=${registry_server} --docker-username=${registry_username} --docker-password=${registry_password} --docker-email=${registry_email}
```

## Example Usage (Service account token)

```hcl
resource "kubernetes_secret_v1" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = "my-service-account"
    }
  }

  type = "kubernetes.io/service-account-token"
}
```

## Argument Reference

The following arguments are supported:

* `data` - (Optional) A map of the secret data.
* `binary_data` - (Optional) A map base64 encoded map of the secret data.
* `metadata` - (Required) Standard secret's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `type` - (Optional) The secret type. Defaults to `Opaque`. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/c7151dd8dd7e487e96e5ce34c6a416bb3b037609/contributors/design-proposals/auth/secrets.md#proposed-design)
* `immutable` - (Optional) Ensures that data stored in the Secret cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time.
* `wait_for_service_account_token` - (Optional) Terraform will wait for the service account token to be created. Defaults to `true`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the secret that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the secret. May match selectors of replication controllers and services.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the secret, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the secret must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this secret that can be used by clients to determine when secret has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this secret. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### Timeouts

`kubernetes_secret_v1` provides the following configuration options:

- `create` - Default `1 minute`

## Import

Secret can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_secret_v1.example default/my-secret
```
