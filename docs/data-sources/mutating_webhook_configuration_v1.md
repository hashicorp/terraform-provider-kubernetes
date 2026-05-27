---
subcategory: "admissionregistration/v1"
page_title: "Kubernetes: kubernetes_mutating_webhook_configuration_v1"
description: |-
  Mutating Webhook Configuration configures a mutating admission webhook
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
data "kubernetes_mutating_webhook_configuration_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}
```
