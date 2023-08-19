---
subcategory: "rbac/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_cluster_role_binding"
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
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }
  subject {
    kind      = "User"
    name      = "admin"
    api_group = "rbac.authorization.k8s.io"
  }
  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = "kube-system"
  }
  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard kubernetes metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `role_ref` - (Required) The ClusterRole to bind Subjects to. For more info see [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding)
* `subject` - (Required) The Users, Groups, or ServiceAccounts to grant permissions to. For more info see [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#referring-to-subjects)


## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the cluster role binding that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the cluster role binding.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the cluster role binding, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this object that can be used by clients to determine when the object has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this cluster role binding. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `role_ref`

#### Arguments

* `name` - (Required) The name of this ClusterRole to bind Subjects to.
* `kind` - (Required) The type of binding to use. This value must be and defaults to `ClusterRole`
* `api_group` - (Required) The API group to drive authorization decisions. This value must be and defaults to `rbac.authorization.k8s.io`

### `subject`

#### Arguments

* `name` - (Required) The name of this ClusterRole to bind Subjects to.
* `namespace` - (Optional) Namespace defines the namespace of the ServiceAccount to bind to. This value only applies to kind `ServiceAccount`
* `kind` - (Required) The type of binding to use. This value must be `ServiceAccount`, `User` or `Group`
* `api_group` - (Required) The API group to drive authorization decisions. This value only applies to kind `User` and `Group`. It must be `rbac.authorization.k8s.io`

## Import

ClusterRoleBinding can be imported using the name, e.g.

```
$ terraform import kubernetes_cluster_role_binding.example terraform-name
```
