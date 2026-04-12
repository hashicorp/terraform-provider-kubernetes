---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_gateway_v1"
description: |-
  Gateway represents an instance of a service-traffic handling infrastructure by binding Listeners to a set of IP addresses.
---

# kubernetes_gateway_v1 (Data Source)

Gateway represents an instance of a service-traffic handling infrastructure by binding Listeners to a set of IP addresses.

## Example Usage

```hcl
data "kubernetes_gateway_v1" "example" {
  metadata {
    name      = "example-gateway"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard gateway's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the gateway.

Optional:

- `namespace` (String) Namespace of the gateway. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of Gateway.
- `status` (Block List) Status defines the current state of Gateway.
