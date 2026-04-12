---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_grpc_route_v1"
description: |-
  GRPCRoute provides a way to route gRPC requests.
---

# kubernetes_grpc_route_v1

GRPCRoute provides a way to route gRPC requests.

## Example Usage

```hcl
resource "kubernetes_grpc_route_v1" "example" {
  metadata {
    name      = "example-grpc-route"
    namespace = "default"
  }
  spec {
    parent_refs {
      name = "example-gateway"
    }
    rules {
      matches {
        method {
          service = "example.v1.Service"
          method  = "GetUser"
        }
      }
      backend_refs {
        name = "grpc-backend"
        port = 50051
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard route's metadata.
- `spec` (Block List, Required) Spec defines the desired state of GRPCRoute.
- `timeouts` (Block, Optional) Standard resource's timeouts.

## Import

`kubernetes_grpc_route_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_grpc_route_v1.example default/example-grpc-route
```