---
subcategory: "scheduling/v1"
page_title: "Kubernetes: kubernetes_priority_class"
description: |-
  A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.
---

# <no value>

A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.

<no value>

## Example Usage

```terraform
resource "kubernetes_priority_class" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
```

## Import

Priority Class can be imported using its name, e.g.

```
$ terraform import kubernetes_priority_class.example terraform-example
```
