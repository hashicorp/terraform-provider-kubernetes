---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_backend_tls_policy_v1"
description: |-
  BackendTLSPolicy configures TLS settings for backend services.
---

# kubernetes_backend_tls_policy_v1

BackendTLSPolicy configures TLS settings for backend services.

## Example Usage

```hcl
resource "kubernetes_backend_tls_policy_v1" "example" {
  metadata {
    name      = "example-backend-tls-policy"
    namespace = "default"
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = "backend-service"
      port  = 443
    }
    tls {
      min_version = "TLS12"
      max_version = "TLS13"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard policy's metadata.
- `spec` (Block List, Required) Spec defines the desired state of BackendTLSPolicy.
- `timeouts` (Block, Optional) Standard resource's timeouts.

### Nested Schema for `spec`

Required:

- `target_refs` (Block List) TargetRefs references the resources to which the policy applies.

Optional:

- `tls` (Block, Max: 1) TLS configuration for the backend.

### Nested Schema for `spec.target_refs`

Required:

- `group` (String) Group of the target resource.
- `kind` (String) Kind of the target resource.
- `name` (String) Name of the target resource.
- `port` (Number) Port of the target resource.

### Nested Schema for `spec.tls`

Optional:

- `min_version` (String) Minimum TLS version. Valid values: TLS10, TLS11, TLS12, TLS13.
- `max_version` (String) Maximum TLS version.
- `certificate_refs` (Block List) Certificate references for client certificate authentication.

## Import

`kubernetes_backend_tls_policy_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_backend_tls_policy_v1.example default/example-backend-tls-policy
```