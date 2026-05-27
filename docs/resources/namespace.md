---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_namespace"
description: |-
  Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces.
---

# <no value> 

Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces. Read more about namespaces at [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

<no value>

## Example Usage

```terraform
resource "kubernetes_namespace" "example" {
  metadata {
    annotations = {
      name = "example-annotation"
    }

    labels = {
      mylabel = "label-value"
    }

    name = "terraform-example-namespace"
  }
}
```

### Timeouts

`kubernetes_namespace` provides the following [Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `delete` - Default `5 minutes`

## Import

Namespaces can be imported using their name, e.g.

```
$ terraform import kubernetes_namespace.n terraform-example-namespace
```
