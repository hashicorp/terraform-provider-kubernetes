---
subcategory: "admissionregistration/v1"
page_title: "Kubernetes: kubernetes_validating_webhook_configuration_v1"
description: |-
  Validating Webhook Configuration configures a validating admission webhook
---

# <no value>

<no value>

<no value> 

## Example Usage

```terraform
resource "kubernetes_validating_webhook_configuration_v1" "example" {
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

    side_effects = "None"
  }
}
```

## API version support

The provider supports clusters running either `v1` or `v1beta1` of the Admission Registration API.

## Import

Validating Webhook Configuration can be imported using the name, e.g.

```
$ terraform import kubernetes_validating_webhook_configuration_v1.example terraform-example
```
