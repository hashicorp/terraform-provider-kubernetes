---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_secret_v1"
description: |-
  Reads a Secret without storing its values in the Terraform plan or state.
---

# Ephemeral: kubernetes_secret_v1

Reads a Secret without storing its values in the Terraform plan or state. Use it in place of the `kubernetes_secret_v1` data source when a value is only passed on, for example to a provider configuration or to a write-only argument.

~> Ephemeral resources are available in Terraform 1.10 and later.

## Example Usage

```terraform
ephemeral "kubernetes_secret_v1" "deployer" {
  metadata {
    name      = "deployer-token"
    namespace = "kube-system"
  }
}

# The token and the CA certificate are used here and stored nowhere.
provider "kubernetes" {
  alias                  = "deployer"
  host                   = var.cluster_endpoint
  token                  = ephemeral.kubernetes_secret_v1.deployer.data["token"]
  cluster_ca_certificate = ephemeral.kubernetes_secret_v1.deployer.data["ca.crt"]
}
```

### A Secret created in the same configuration

Terraform reads an ephemeral resource while planning whenever its arguments are known then, and a Secret that does not exist yet is an error. To read a Secret that the same configuration creates, build the name from the `id` of that resource. The `id` is not known until the Secret exists, so Terraform waits for it.

```terraform
resource "kubernetes_secret_v1" "token" {
  metadata {
    name      = "deployer-token"
    namespace = "kube-system"
    annotations = {
      "kubernetes.io/service-account.name" = "deployer"
    }
  }
  type                           = "kubernetes.io/service-account-token"
  wait_for_service_account_token = true
}

ephemeral "kubernetes_secret_v1" "token" {
  metadata {
    namespace = split("/", kubernetes_secret_v1.token.id)[0]
    name      = split("/", kubernetes_secret_v1.token.id)[1]
  }
}
```

## Schema

### Required

- `metadata` (Block) The Secret to read. (see [below for nested schema](#nestedblock--metadata))

### Read-Only

- `binary_data` (Map of String, Sensitive) Every value of the Secret in base64 encoding, by key.
- `data` (Map of String, Sensitive) The values of the Secret that are valid UTF-8 text, by key. A value that is not text is only in `binary_data`.
- `immutable` (Boolean) Whether the Secret is marked immutable.
- `type` (String) The type of the Secret, such as `Opaque` or `kubernetes.io/tls`.

<a id="nestedblock--metadata"></a>
### Nested Schema for `metadata`

Required:

- `name` (String) Name of the Secret.

Optional:

- `namespace` (String) Namespace of the Secret. Defaults to `default`.
