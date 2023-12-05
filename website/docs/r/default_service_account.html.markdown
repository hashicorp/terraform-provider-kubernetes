---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_default_service_account"
description: |-
  The default service account resource configures the default service account created by Kubernetes in each namespace.
---

# kubernetes_default_service_account

Kubernetes creates a "default" service account in each namespace. This is the service account that will be assigned by default to pods in the namespace. 

The `kubernetes_default_service_account` resource behaves differently from normal resources. The service account is created by a Kubernetes controller and Terraform "adopts" it into management. This resource should only be used once per namespace.

## Example Usage

```hcl
resource "kubernetes_default_service_account" "example" {
  metadata {
    namespace = "terraform-example"
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
* `image_pull_secret` - (Optional) A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod)
* `secret` - (Optional) A list of secrets allowed to be used by pods running using this Service Account. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/configuration/secret)
* `automount_service_account_token` - (Optional) Boolean, `true` to enable automatic mounting of the service account token. Defaults to `true`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the service account that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service account. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `namespace` - (Optional) Namespace defines the namespace where Terraform will adopt the default service account.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service account that can be used by clients to determine when service account has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this service account. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `image_pull_secret`

#### Arguments

* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `secret`

#### Arguments

* `name` - (Optional) Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `default_secret_name` - (Deprecated) Name of the default secret, containing service account token, created & managed by the service. By default, the provider will try to find the secret containing the service account token that Kubernetes automatically created for the service account. Where there are multiple tokens and the provider cannot determine which was created by Kubernetes, this attribute will be empty. When only one token is associated with the service account, the provider will return this single token secret.

  Starting from version `1.24.0` by default Kubernetes does not automatically generate tokens for service accounts. That leads to the situation when `default_secret_name` cannot be computed and thus will be an empty string. In order to create a service account token, please [use `kubernetes_secret_v1` resource](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/secret_v1#example-usage-service-account-token)

## Destroying

If you remove a `kubernetes_default_service_account` resource from your configuration, Terraform will send a delete request to the Kubernetes API. Kubernetes will automatically replace this service account, but any customizations will be lost. If you no longer want to manage a default service account with Terraform, use `terraform state rm` to remove it from state before removing the configuration.

## Import

The default service account can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_default_service_account.example terraform-example/default
```
