---
subcategory: "networking/v1"
page_title: "Kubernetes: kubernetes_ingress_class"
description: |-
  Ingresses can be implemented by different controllers, often with different configuration. Each Ingress should specify a class, a reference to an IngressClass resource that contains additional configuration including the name of the controller that should implement the class.
---

# <no value>

Ingresses can be implemented by different controllers, often with different configuration. Each Ingress should specify a class, a reference to an IngressClass resource that contains additional configuration including the name of the controller that should implement the class.

<no value>

## Example Usage

```terraform
resource "kubernetes_ingress_class" "example" {
  metadata {
    name = "example"
  }

  spec {
    controller = "example.com/ingress-controller"
    parameters {
      api_group = "k8s.example.com"
      kind      = "IngressParameters"
      name      = "external-lb"
    }
  }
}
```

## Import

Ingress Classes can be imported using its name, e.g:

```
$ terraform import kubernetes_ingress_class.example example
```
