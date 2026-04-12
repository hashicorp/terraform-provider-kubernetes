---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_http_route_v1"
description: |-
  HTTPRoute provides a way to route HTTP requests.
---

# kubernetes_http_route_v1 (Data Source)

HTTPRoute provides a way to route HTTP requests. This matches HTTP/1.1 and HTTP/2 traffic.

## Example Usage

```hcl
data "kubernetes_http_route_v1" "example" {
  metadata {
    name      = "example-httproute"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard HTTP route's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the HTTPRoute.

Optional:

- `namespace` (String) Namespace of the HTTPRoute. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of HTTPRoute.
- `status` (Block List) Status defines the current state of HTTPRoute.
