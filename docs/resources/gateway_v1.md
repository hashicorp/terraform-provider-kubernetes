---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_gateway_v1"
description: |-
  Gateway represents an instance of a service-traffic handling infrastructure by binding Listeners to a set of IP addresses.
---

# kubernetes_gateway_v1

Gateway represents an instance of a service-traffic handling infrastructure by binding Listeners to a set of IP addresses.

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

resource "kubernetes_gateway_v1" "example" {
  metadata {
    name      = "example-gateway"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.example.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
    }
    listeners {
      name     = "https"
      port     = 443
      protocol = "HTTPS"
      tls {
        mode = "Terminate"
        certificate_refs {
          name = "example-tls-secret"
          kind = "Secret"
        }
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard gateway's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
- `spec` (Block List, Required) Spec defines the desired state of Gateway. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
- `timeouts` (Block, Optional) Standard resource's timeouts. More info: https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/timeouts

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the gateway, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names

Optional:

- `namespace` (String) Namespace defines the space within which name of the gateway must be unique.
- `labels` (Map of String) Map of string keys and values that can be used to organize and categorize the gateway.
- `annotations` (Map of String) An unstructured key value map stored with the gateway.

### Nested Schema for `spec`

Required:

- `gateway_class_name` (String) GatewayClassName used for this Gateway. This is the name of a GatewayClass resource.
- `listeners` (Block List, Min: 1) Listeners associated with this Gateway.

Optional:

- `addresses` (Block List) Addresses requested for this Gateway.

### Nested Schema for `spec.listeners`

Required:

- `name` (String) Name is the name of the Listener.
- `port` (Number) Port is the network port. Valid values are 1-65535.
- `protocol` (String) Protocol specifies the network protocol. Valid values: HTTP, HTTPS, TCP, UDP, TLS, GRPC.

Optional:

- `hostname` (String) Hostname specifies the virtual hostname to match for protocol types that define this concept.
- `tls` (Block List, Max: 1) TLS is the TLS configuration for the Listener.
- `allowed_routes` (Block List, Max: 1) AllowedRoutes specifies the namespaces and routes that may be used by this Listener.

### Nested Schema for `spec.listeners.tls`

Optional:

- `mode` (String) Mode specifies the TLS mode. Valid values: Terminate, Passthrough.
- `certificate_refs` (Block List) Certificate refs for TLS.

### Nested Schema for `spec.listeners.allowed_routes`

Optional:

- `namespaces` (Block List, Max: 1) Namespaces specifies which namespaces may be used as targets for this Listener.
- `kinds` (Block List) Kinds specifies the resource kinds that may be referenced.

## Import

`kubernetes_gateway_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_gateway_v1.example default/example-gateway
```