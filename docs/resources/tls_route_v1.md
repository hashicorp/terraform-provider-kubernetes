---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_tls_route_v1"
description: |-
  TLSRoute provides a way to route TLS requests.
---

# kubernetes_tls_route_v1

TLSRoute provides a way to route TLS requests.

## Example Usage

```hcl
resource "kubernetes_tls_route_v1" "example" {
  metadata {
    name      = "example-tls-route"
    namespace = "default"
  }
  spec {
    parent_refs {
      name = "example-gateway"
    }
    rules {
      backend_refs {
        name = "tls-backend"
        port = 443
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard route's metadata.
- `spec` (Block List, Required) Spec defines the desired state of TLSRoute.
- `timeouts` (Block, Optional) Standard resource's timeouts.

## Import

`kubernetes_tls_route_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_tls_route_v1.example default/example-tls-route
```