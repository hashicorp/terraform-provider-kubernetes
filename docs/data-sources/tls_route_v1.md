---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_tls_route_v1"
description: |-
  TLSRoute provides a way to route TLS requests.
---

# kubernetes_tls_route_v1 (Data Source)

TLSRoute provides a way to route TLS requests. This is used with TLS-passthrough Gateways.

## Example Usage

```hcl
data "kubernetes_tls_route_v1" "example" {
  metadata {
    name      = "example-tlsroute"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard TLS route's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the TLSRoute.

Optional:

- `namespace` (String) Namespace of the TLSRoute. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of TLSRoute.
- `status` (Block List) Status defines the current state of TLSRoute.
