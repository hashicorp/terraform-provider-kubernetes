---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_gateway_class_v1"
description: |-
  GatewayClass represents a class of Gateways available to the user for creating Gateway resources.
---

# kubernetes_gateway_class_v1 (Data Source)

GatewayClass represents a class of Gateways available to the user for creating Gateway resources.

## Example Usage

```hcl
data "kubernetes_gateway_class_v1" "example" {
  metadata {
    name = "example-gateway-class"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard gateway class's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the gateway class.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of GatewayClass.
- `status` (Block List) Status defines the current state of GatewayClass.
