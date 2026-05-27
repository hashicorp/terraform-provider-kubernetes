---
subcategory: "scheduling/v1"
page_title: "Kubernetes: kubernetes_priority_class_v1"
description: |-
  A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_priority_class_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  value = 100
}
```

## Import

Priority Class can be imported using its name, e.g.

```
$ terraform import kubernetes_priority_class_v1.example terraform-example
```
