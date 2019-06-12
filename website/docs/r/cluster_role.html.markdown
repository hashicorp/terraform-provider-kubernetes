---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_cluster_role"
sidebar_current: "docs-kubernetes-resource-cluster-role"
description: |-
  A ClusterRole creates a role at the cluster level and in all namespaces.
---

# kubernetes_cluster_role

A ClusterRole creates a role at the cluster level and in all namespaces.

## Example Usage

```hcl
resource "kubernetes_cluster_role" "example" {
  metadata {
    name = "terraform-example"
  }

  rule {
    api_groups = [""]
    resources  = ["namespaces", "pods"]
    verbs      = ["get", "list", "watch"]
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` - (Required) Standard kubernetes metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata)
- `rule` - (Required) The PolicyRoles for this ClusterRole. For more info see [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)

## Nested Blocks

### `metadata`

#### Arguments

- `annotations` - (Optional) An unstructured key value map stored with the cluster role binding that may be used to store arbitrary metadata. 
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
- `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency)
- `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the cluster role binding. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
- `name` - (Optional) Name of the cluster role binding, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

#### Attributes

- `generation` - A sequence number representing a specific generation of the desired state.
- `resource_version` - An opaque value that represents the internal version of this object that can be used by clients to determine when the object has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
- `self_link` - A URL representing this cluster role binding.
- `uid` - The unique in time and space value for this cluster role binding. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `rule`

#### Arguments

- `api_groups` - (Optional) APIGroups is the name of the APIGroup that contains the resources. If multiple API groups are specified, any action requested against one of the enumerated resources in any API group will be allowed.
- `non_resource_urls` - (Optional) NonResourceURLs is a set of partial urls that a user should have access to. \*s are allowed, but only as the full, final step in the path Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding. Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"), but not both.
- `resource_names` - (Optional) ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed.
- `resources` - (Optional) Resources is a list of resources this rule applies to. ResourceAll represents all resources.
- `verbs` - (Required) Verbs is a list of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule. VerbAll represents all kinds.

## Import

ClusterRole can be imported using the name, e.g.

```
$ terraform import kubernetes_cluster_role.example terraform-name
```
