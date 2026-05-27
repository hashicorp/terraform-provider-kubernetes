---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_default_service_account_v1"
description: |-
  The default service account resource configures the default service account created by Kubernetes in each namespace.
---

# <no value>

Kubernetes creates a "default" service account in each namespace. This is the service account that will be assigned by default to pods in the namespace.

The `kubernetes_default_service_account_v1` resource behaves differently from normal resources. The service account is created by a Kubernetes controller and Terraform "adopts" it into management. This resource should only be used once per namespace.

<no value>

## Example Usage

```terraform
resource "kubernetes_default_service_account_v1" "example" {
  metadata {
    namespace = "terraform-example"
  }
  secret {
    name = "${kubernetes_secret_v1.example.metadata.0.name}"
  }
}

resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}
```

## Import

The default service account can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_default_service_account_v1.example terraform-example/default
```
