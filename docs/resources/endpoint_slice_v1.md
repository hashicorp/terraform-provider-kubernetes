---
subcategory: "discovery/v1"
page_title: "Kubernetes: kubernetes_endpoint_slice_v1"
description: |-
  An EndpointSlice contains references to a set of network endpoints.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "test"
  }

  endpoint {
    condition {
      ready = true
    }
    addresses = ["129.144.50.56"]
  }

  port {
    port = "9000"
    name = "first"
  }

  address_type = "IPv4"
}
```

