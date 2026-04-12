---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_grpc_route_v1"
description: |-
  GRPCRoute provides a way to route gRPC requests.
---

# kubernetes_grpc_route_v1 (Data Source)

GRPCRoute provides a way to route gRPC requests. This matches gRPC traffic by method and service.

## Example Usage

```hcl
data "kubernetes_grpc_route_v1" "example" {
  metadata {
    name      = "example-grpcroute"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard GRPC route's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the GRPCRoute.

Optional:

- `namespace` (String) Namespace of the GRPCRoute. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of GRPCRoute.
- `status` (Block List) Status defines the current state of GRPCRoute.
