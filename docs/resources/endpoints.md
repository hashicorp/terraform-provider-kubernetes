---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_endpoints"
description: |-
  An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.
---

# <no value> 

An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.

<no value>

## Example Usage

```terraform
resource "kubernetes_endpoints" "example" {
  metadata {
    name = "terraform-example"
  }

  subset {
    address {
      ip = "10.0.0.4"
    }

    address {
      ip = "10.0.0.5"
    }

    port {
      name     = "http"
      port     = 80
      protocol = "TCP"
    }

    port {
      name     = "https"
      port     = 443
      protocol = "TCP"
    }
  }

  subset {
    address {
      ip = "10.0.1.4"
    }

    address {
      ip = "10.0.1.5"
    }

    port {
      name     = "http"
      port     = 80
      protocol = "TCP"
    }

    port {
      name     = "https"
      port     = 443
      protocol = "TCP"
    }
  }
}

resource "kubernetes_service" "example" {
  metadata {
    name = "${kubernetes_endpoints.example.metadata.0.name}"
  }

  spec {
    port {
      port        = 8080
      target_port = 80
    }

    port {
      port        = 8443
      target_port = 443
    }
  }
}
```

## Import

An Endpoints resource can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_endpoints.example default/terraform-name
```
