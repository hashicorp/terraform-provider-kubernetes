---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_resource_quota"
description: |-
  A resource quota provides constraints that limit aggregate resource consumption per namespace. It can limit the quantity of objects that can be created in a namespace by type, as well as the total amount of compute resources that may be consumed by resources in that project.
---

# <no value>

A resource quota provides constraints that limit aggregate resource consumption per namespace. It can limit the quantity of objects that can be created in a namespace by type, as well as the total amount of compute resources that may be consumed by resources in that project.

<no value>

## Example Usage

```terraform
resource "kubernetes_resource_quota" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    hard = {
      pods = 10
    }
    scopes = ["BestEffort"]
  }
}
```

## Import

Resource Quota can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_resource_quota.example default/terraform-example
```
