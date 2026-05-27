---
subcategory: "admissionregistration/v1beta1"
page_title: "Kubernetes: kubernetes_mutating_webhook_configuration"
description: |-
  Mutating Webhook Configuration configures a mutating admission webhook
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_mutating_webhook_configuration" "example" {
  metadata {
    name = "test.terraform.io"
  }

  webhook {
    name = "test.terraform.io"

    admission_review_versions = ["v1", "v1beta1"]

    client_config {
      service {
        namespace = "example-namespace"
        name      = "example-service"
      }
    }

    rule {
      api_groups   = ["apps"]
      api_versions = ["v1"]
      operations   = ["CREATE"]
      resources    = ["deployments"]
      scope        = "Namespaced"
    }

    reinvocation_policy = "IfNeeded"
    side_effects        = "None"
  }
}
```

## API version support

The provider supports clusters running either `v1` or `v1beta1` of the Admission Registration API.

## Import

Mutating Webhook Configuration can be imported using the name, e.g.

```
$ terraform import kubernetes_mutating_webhook_configuration.example terraform-example
```
