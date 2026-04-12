---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_listener_set_v1"
description: |-
  ListenerSet defines a set of additional listeners to attach to an existing Gateway.
---

# kubernetes_listener_set_v1 (Data Source)

ListenerSet defines a set of additional listeners to attach to an existing Gateway.

## Example Usage

```hcl
data "kubernetes_listener_set_v1" "example" {
  metadata {
    name      = "example-listenerset"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard listener set's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the ListenerSet.

Optional:

- `namespace` (String) Namespace of the ListenerSet. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of ListenerSet.
- `status` (Block List) Status defines the current state of ListenerSet.
