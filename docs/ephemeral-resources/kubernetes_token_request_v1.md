---
subcategory: "authentication/v1"
page_title: "Kubernetes: kubernetes_token_request_v1"
description: |-
  TokenRequest requests a token for a given service account.
---

# Ephemeral: kubernetes_token_request_v1

TokenRequest requests a token for a given service account.

## Schema

### Required

- `metadata` (Block List, Min: 1, Max: 1) Standard token request's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata (see [below for nested schema](#nestedblock--metadata))

### Optional

- `spec` (Block List, Max: 1) (see [below for nested schema](#nestedblock--spec))

### Read-Only

- `token` (String, Sensitive) Token is the opaque bearer token.

<a id="nestedblock--metadata"></a>
### Nested Schema for `metadata`

Optional:

- `name` (String) Name of the token request, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
- `namespace` (String) Namespace defines the space within which name of the token request must be unique.

<a id="nestedblock--spec"></a>
### Nested Schema for `spec`

Optional:

- `audiences` (List of String) Audiences are the intendend audiences of the token. A recipient of a token must identify themself with an identifier in the list of audiences of the token, and otherwise should reject the token. A token issued for multiple audiences may be used to authenticate against any of the audiences listed but implies a high degree of trust between the target audiences.
- `bound_object_ref` (Block List, Max: 1) BoundObjectRef is a reference to an object that the token will be bound to. The token will only be valid for as long as the bound object exists. NOTE: The API server's TokenReview endpoint will validate the BoundObjectRef, but other audiences may not. Keep ExpirationSeconds small if you want prompt revocation. (see [below for nested schema](#nestedblock--spec--bound_object_ref))
- `expiration_seconds` (Number) expiration_seconds is the requested duration of validity of the request. The token issuer may return a token with a different validity duration so a client needs to check the 'expiration' field in a response. The expiration can't be less than 10 minutes.

<a id="nestedblock--spec--bound_object_ref"></a>
### Nested Schema for `spec.bound_object_ref`

Optional:

- `api_version` (String) API version of the referent.
- `kind` (String) Kind of the referent. Valid kinds are 'Pod' and 'Secret'.
- `name` (String) Name of the referent.
- `uid` (String) UID of the referent.

## Example Usage

```terraform
resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "test"
  }
}

ephemeral "kubernetes_token_request_v1" "test" {
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
```

