---
subcategory: "apiregistration/v1"
page_title: "Kubernetes: kubernetes_api_service"
description: |-
  An API Service is an abstraction which defines for locating and communicating with servers.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_api_service" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    selector {
      app = "${kubernetes_pod.example.metadata.0.labels.app}"
    }
    session_affinity = "ClientIP"
    port {
      port        = 8080
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
```

## Import

API service can be imported using its name, e.g.

```
$ terraform import kubernetes_api_service.example v1.terraform-name.k8s.io
```
