---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_service_account"
description: |-
  A service account provides an identity for processes that run in a Pod.
---

# <no value>

A service account provides an identity for processes that run in a Pod.

Read more at [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/)

<no value>

## Example Usage

```terraform
resource "kubernetes_service_account" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "kubernetes_secret" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = kubernetes_service_account.example.metadata.0.name
    }

    generate_name = "terraform-example-"
  }

  type                           = "kubernetes.io/service-account-token"
  wait_for_service_account_token = true
}
```

## Import

Service account can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_service_account.example default/terraform-example
```
