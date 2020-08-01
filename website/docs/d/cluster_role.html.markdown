---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_cluster_role"
sidebar_current: "docs-kubernetes-data-source-cluster-role"
description: |-
  A ClusterRole creates a role at the cluster level and in all namespaces.
---

# kubernetes_cluster_role

A ClusterRole creates a role at the cluster level and in all namespaces.

## Example Usage

```hcl
data "kubernetes_cluster_role" "example" {
  metadata {
    name = "terraform-example"
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` - (Required) Standard kubernetes metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

- `name` - (Optional) Name of the cluster role binding, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

#### Attributes

- `generation` - A sequence number representing a specific generation of the desired state.
- `resource_version` - An opaque value that represents the internal version of this object that can be used by clients to determine when the object has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
- `self_link` - A URL representing this cluster role binding.
- `uid` - The unique in time and space value for this cluster role binding. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `rules`

#### Attributes

- `api_groups` - APIGroups is the name of the APIGroup that contains the resources.
- `non_resource_urls` - NonResourceURLs is a set of partial urls that a user should have access to.
- `resource_names` - ResourceNames is a list of names that the rule applies to. An empty set means that everything is allowed.
- `resources` - Resources is a list of resources this rule applies to. ResourceAll represents all resources.
- `verbs` - Verbs is a list of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule. VerbAll represents all kinds.
