---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_role"
sidebar_current: "docs-kubernetes-resource-role"
description: |-
  A role contains rules that represent a set of permissions. Permissions are purely additive (there are no “deny” rules).
---

# kubernetes_role

A role contains rules that represent a set of permissions. Permissions are purely additive (there are no “deny” rules).


## Example Usage

```hcl
resource "kubernetes_role" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      test = "MyRole"
    }
  }

  rule {
    api_groups     = [""]
    resources      = ["pods"]
    resource_names = ["foo"]
    verbs          = ["get", "list", "watch"]
  }
  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard role's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `rule` - (Required) List of rules that define the set of permissions for this role. For more info see [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the role that may be used to store arbitrary metadata.
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](hhttps://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the role. **Must match `selector`**.
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the role, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the role must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this role that can be used by clients to determine when role has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this role.
* `uid` - The unique in time and space value for this role. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `rule`

#### Arguments

* `api_groups` - (Required) List of APIGroups that contains the resources.
* `resources` - (Required) List of resources that the rule applies to.
* `resource_names` - (Optional) White list of names that the rule applies to.
* `verbs` - (Required) List of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule.

## Import

Role can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_role.example default/terraform-example
```
