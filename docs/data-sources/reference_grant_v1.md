---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_reference_grant_v1"
description: |-
  ReferenceGrant allows cross-namespace references in Gateway API resources.
---

# kubernetes_reference_grant_v1 (Data Source)

ReferenceGrant allows cross-namespace references in Gateway API resources.

## Example Usage

```hcl
data "kubernetes_reference_grant_v1" "example" {
  metadata {
    name      = "example-referencegrant"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard reference grant's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the ReferenceGrant.

Optional:

- `namespace` (String) Namespace of the ReferenceGrant. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of ReferenceGrant.
