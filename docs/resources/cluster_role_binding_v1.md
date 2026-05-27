---
subcategory: "rbac/v1"
page_title: "Kubernetes: kubernetes_cluster_role_binding_v1"
description: |-
  A ClusterRoleBinding may be used to grant permission at the cluster level and in all namespaces.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_cluster_role_binding_v1" "example" {
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

## Import

ClusterRoleBinding can be imported using the name, e.g.

```
$ terraform import kubernetes_cluster_role_binding_v1.example terraform-name
```
