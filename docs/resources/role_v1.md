---
subcategory: "rbac/v1"
page_title: "Kubernetes: kubernetes_role_v1"
description: |-
  A role contains rules that represent a set of permissions. Permissions are purely additive (there are no “deny” rules).
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_role_v1" "example" {
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

## Import

Role can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_role_v1.example default/terraform-example
```
