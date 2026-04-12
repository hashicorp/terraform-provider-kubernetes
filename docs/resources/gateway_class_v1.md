---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_gateway_class_v1"
description: |-
  GatewayClass represents a class of Gateways available to the user for creating Gateway resources.
---

# kubernetes_gateway_class_v1

GatewayClass represents a class of Gateways available to the user for creating Gateway resources.

## Example Usage

```hcl
resource "kubernetes_gateway_class_v1" "example" {
  metadata {
    name = "example-gateway-class"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard gateway class's metadata.
- `spec` (Block List, Required) Spec defines the desired state of GatewayClass.
- `timeouts` (Block, Optional) Standard resource's timeouts.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the gateway class, must be unique.

Optional:

- `labels` (Map of String) Map of string keys and values.
- `annotations` (Map of String) An unstructured key value map.

### Nested Schema for `spec`

Required:

- `controller_name` (String) ControllerName specifies the name of the controller that manages Gateways of this class.

Optional:

- `description` (String) Description of the GatewayClass.

## Import

`kubernetes_gateway_class_v1` can be imported using the name, e.g.

```
$ terraform import kubernetes_gateway_class_v1.example example-gateway-class
```