---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_service_account"
description: |-
  A service account provides an identity for processes that run in a Pod.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
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
