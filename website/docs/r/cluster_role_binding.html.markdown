---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_cluster_role_binding"
sidebar_current: "docs-kubernetes-resource-cluster-role-binding"
description: |-
  A ClusterRoleBinding may be used to grant permission at the cluster level and in all namespaces.
---

# kubernetes_cluster_role_binding

A ClusterRoleBinding may be used to grant permission at the cluster level and in all namespaces


## Example Usage

```hcl
resource "kubernetes_cluster_role_binding" "example" {
	metadata {
		name = "terraform-example"
	}
	role_ref {
		api_group = "rbac.authorization.k8s.io"
		kind = "ClusterRole"
		name = "cluster-admin"
	}
	subject {
		kind = "User"
		name = "admin"
		api_group = "rbac.authorization.k8s.io"
	}
	subject {
		kind = "ServiceAccount"
		name = "default"
		namespace = "kube-system"
	}
	subject {
		kind = "Group"
		name = "system:masters"
		api_group = "rbac.authorization.k8s.io"
	}
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard kubernetes metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* `role_ref` - (Required) The ClusterRole to bind Subjects to. More info: https://kubernetes.io/docs/admin/authorization/rbac/#rolebinding-and-clusterrolebinding
* `subject` - (Required) The Users, Groups, or ServiceAccounts to grand permissions to. More info: https://kubernetes.io/docs/admin/authorization/rbac/#referring-to-subjects


## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the cluster role binding that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the cluster role binding. More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the cluster role binding, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this object that can be used by clients to determine when the object has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `self_link` - A URL representing this cluster role binding.
* `uid` - The unique in time and space value for this cluster role binding. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

### `role_ref`

#### Arguments

* `name` - (Required) The name of this ClusterRole to bind Subjects to.
* `kind` - (Required) The type of binding to use. This value must be and defaults to `ClusterRole`
* `api_group` - (Optional) The API group to drive authorization decisions. This value must be and defaults to `rbac.authorization.k8s.io`

### `subject`

#### Arguments

* `name` - (Required) The name of this ClusterRole to bind Subjects to.
* `namespace` - (Optional) Namespace defines the namespace of the ServiceAccount to bind to. This value only applies to kind `ServiceAccount`
* `kind` - (Required) The type of binding to use. This value must be `ServiceAccount`, `User` or `Group`
* `api_group` - (Optional) The API group to drive authorization decisions. This value only applies to kind `User` and `Group`. It must be `rbac.authorization.k8s.io`

## Import

ClusterRoleBinding can be imported using the name, e.g.

```
$ terraform import kubernetes_cluster_role_binding.example terraform-name
```
