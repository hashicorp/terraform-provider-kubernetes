---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_listener_set_v1"
description: |-
  ListenerSet defines a set of additional listeners to attach to an existing Gateway.
---

# kubernetes_listener_set_v1

ListenerSet defines a set of additional listeners to attach to an existing Gateway.

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

resource "kubernetes_listener_set_v1" "example" {
  metadata {
    name      = "example-listener-set"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.example.metadata.0.name
    listeners {
      name     = "http-alt"
      port     = 8080
      protocol = "HTTP"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard listener set's metadata.
- `spec` (Block List, Required) Spec defines the desired state of ListenerSet.
- `timeouts` (Block, Optional) Standard resource's timeouts.

## Import

`kubernetes_listener_set_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_listener_set_v1.example default/example-listener-set
```