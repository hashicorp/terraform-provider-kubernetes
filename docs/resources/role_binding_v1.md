---
subcategory: "rbac/v1"
page_title: "Kubernetes: kubernetes_role_binding_v1"
description: |-
  A RoleBinding may be used to grant permission at the namespace level.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_role_binding_v1" "example" {
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

## Import

RoleBinding can be imported using the name, e.g.

```
$ terraform import kubernetes_role_binding_v1.example default/terraform-name
```
