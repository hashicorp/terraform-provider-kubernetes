---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_endpoints_v1"
description: |-
    An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.
---

# <no value>

<no value>

<no value> 

## Example Usage

```terraform
data "kubernetes_endpoints_v1" "api_endpoints" {
  metadata {
    name      = "kubernetes"
    namespace = "default"
  }
}
```

