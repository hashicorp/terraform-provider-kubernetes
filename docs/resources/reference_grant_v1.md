---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_reference_grant_v1"
description: |-
  ReferenceGrant allows cross-namespace references in Gateway API resources.
---

# kubernetes_reference_grant_v1

ReferenceGrant allows cross-namespace references in Gateway API resources.

## Example Usage

```hcl
resource "kubernetes_reference_grant_v1" "example" {
  metadata {
    name      = "example-reference-grant"
    namespace = "default"
  }
  spec {
    from {
      group = "gateway.networking.k8s.io"
      kind  = "HTTPRoute"
      namespace = "gateway-namespace"
    }
    to {
      group = ""
      kind  = "Service"
      name  = "backend-service"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard grant's metadata.
- `spec` (Block List, Required) Spec defines the desired state of ReferenceGrant.
- `timeouts` (Block, Optional) Standard resource's timeouts.

### Nested Schema for `spec`

Required:

- `from` (Block List) From defines the list of contexts from which references may be made.
- `to` (Block List) To defines the list of contexts to which references may be made.

### Nested Schema for `spec.from`

Required:

- `group` (String) Group of the source resource.
- `kind` (String) Kind of the source resource.
- `namespace` (String) Namespace of the source resource.

### Nested Schema for `spec.to`

Required:

- `group` (String) Group of the target resource.
- `kind` (String) Kind of the target resource.

Optional:

- `name` (String) Name of the target resource. Use "*" to allow all.

## Import

`kubernetes_reference_grant_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_reference_grant_v1.example default/example-reference-grant
```