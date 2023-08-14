---
layout: "kubernetes"
subcategory: "authentication/v1"
page_title: "Kubernetes: kubernetes_token_request_v1"
description: |-
  TokenRequest requests a token for a given service account.
---

# kubernetes_token_request_v1

TokenRequest requests a token for a given service account.


## Example Usage

```hcl
resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_token_request_v1" "test" {
  metadata {
    name = kubernetes_service_account_v1.test.metadata.0.name
  }
  spec {
    audiences = [
      "api",
      "vault",
      "factors"
    ]
  }
}

output "tokenValue" {
  value = kubernetes_token_request_v1.test.token
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard role's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Spec holds information about the request being evaluated

### Attributes

* `token` - Token is the opaque bearer token.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the role that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](hhttps://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the role. **Must match `selector`**.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the role, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the role must be unique.

### `spec`

#### Arguments

* `audiences` - (Optional) Audiences are the intendend audiences of the token. A recipient of a token must identify themself with an identifier in the list of audiences of the token, and otherwise should reject the token. A token issued for multiple audiences may be used to authenticate against any of the audiences listed but implies a high degree of trust between the target audiences.
* `expiration_seconds` - (Optional) ExpirationSeconds is the requested duration of validity of the request. The token issuer may return a token with a different validity duration so a client needs to check the 'expiration' field in a response.
* `bound_object_ref` - (Optional) BoundObjectRef is a reference to an object that the token will be bound to. The token will only be valid for as long as the bound object exists. NOTE: The API server's TokenReview endpoint will validate the BoundObjectRef, but other audiences may not. Keep ExpirationSeconds small if you want prompt revocation.

### `bound_object_ref`

#### Arguments

* `api_version` - (Optional) API version of the referent.
* `kind` - (Optional) Kind of the referent. Valid kinds are 'Pod' and 'Secret'.
* `name` - (Optional) Name of the referent.
* `uid` - (Optional) UID of the referent.
