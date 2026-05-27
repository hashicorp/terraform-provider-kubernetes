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
data "kubernetes_service_account_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}

data "kubernetes_secret" "example" {
  metadata {
    name = "${data.kubernetes_service_account_v1.example.default_secret_name}"
  }
}
```

