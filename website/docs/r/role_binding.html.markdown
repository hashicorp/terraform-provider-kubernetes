---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_role_binding"
sidebar_current: "docs-kubernetes-resource-role-binding"
description: |-
  A RoleBinding may be used to grant permission at the namespace level.
---

# kubernetes_role_binding

A RoleBinding may be used to grant permission at the namespace level

## Example Usage

```hcl
resource "kubernetes_role_binding" "example" {
  metadata {
    name      = "terraform-example"
    namespace = "default"
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
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

* `metadata` - (Required) Standard kubernetes metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata)
* `role_ref` - (Required) The Role to bind Subjects to. For more info see [Kubernetes reference](https://kubernetes.io/docs/admin/authorization/rbac/#rolebinding-and-clusterrolebinding)
* `subject` - (Required) The Users, Groups, or ServiceAccounts to grand permissions to. For more info see [Kubernetes reference](https://kubernetes.io/docs/admin/authorization/rbac/#referring-to-subjects)


## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the role binding that may be used to store arbitrary metadata. 
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the role binding. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the role binding, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the role binding must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this object that can be used by clients to determine when the object has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this role binding.
* `uid` - The unique in time and space value for this role binding. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `role_ref`

#### Arguments

* `name` - (Required) The name of this Role to bind Subjects to.
* `kind` - (Required) The type of binding to use. This value must be present and defaults to `Role`
* `api_group` - (Optional) The API group to drive authorization decisions. This value must be and defaults to `rbac.authorization.k8s.io`

### `subject`

#### Arguments

* `name` - (Required) The name of this Role to bind Subjects to.
* `namespace` - (Optional) Namespace defines the namespace of the ServiceAccount to bind to. This value only applies to kind `ServiceAccount`
* `kind` - (Required) The type of binding to use. This value must be `ServiceAccount`, `User` or `Group`
* `api_group` - (Optional) The API group to drive authorization decisions. This value only applies to kind `User` and `Group`. It must be `rbac.authorization.k8s.io`

## Import

RoleBinding can be imported using the name, e.g.

```
$ terraform import kubernetes_role_binding.example default/terraform-name
```
