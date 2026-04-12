---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_backend_tls_policy_v1"
description: |-
  BackendTLSPolicy configures TLS settings for backend services.
---

# kubernetes_backend_tls_policy_v1 (Data Source)

BackendTLSPolicy configures TLS settings for connections to backend services.

## Example Usage

```hcl
data "kubernetes_backend_tls_policy_v1" "example" {
  metadata {
    name      = "example-backendtlspolicy"
    namespace = "default"
  }
}
```

## Argument Reference

- `metadata` (Block List, Required) Standard backend TLS policy's metadata.

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the BackendTLSPolicy.

Optional:

- `namespace` (String) Namespace of the BackendTLSPolicy. Defaults to `default`.

## Attributes Reference

- `spec` (Block List) Spec defines the desired state of BackendTLSPolicy.
- `status` (Block List) Status defines the current state of BackendTLSPolicy.
