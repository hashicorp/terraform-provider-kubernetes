---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_service_account_v1"
description: |-
  A service account provides an identity for processes that run in a Pod.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_service_account_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "kubernetes_secret_v1" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = kubernetes_service_account_v1.example.metadata.0.name
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
$ terraform import kubernetes_service_account_v1.example default/terraform-example
```
