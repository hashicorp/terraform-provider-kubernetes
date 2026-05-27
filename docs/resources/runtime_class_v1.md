---
subcategory: "node/v1"
page_title: "Kubernetes: kubernetes_runtime_class_v1"
description: |-
  A runtime class is used to determine which container runtime is used to run all containers in a pod. 
---

# <no value>

<no value>

<no value>

## Example usage

```terraform
resource "kubernetes_runtime_class_v1" "example" {
  metadata {
    name = "myclass"
  }
  handler = "abcdeagh"
}
```

## Import

Runtime class can be imported using the name only, e.g.

```
$ terraform import kubernetes_runtime_class_v1.example myclass
```
