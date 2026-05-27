---
subcategory: "networking/v1"
page_title: "Kubernetes: kubernetes_ingress_v1"
description: |-
  Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_ingress_v1" "example_ingress" {
  metadata {
    name = "example-ingress"
  }

  spec {
    default_backend {
      service {
        name = "myapp-1"
        port {
          number = 8080
        }
      }
    }

    rule {
      http {
        path {
          backend {
            service {
              name = "myapp-1"
              port {
                number = 8080
              }
            }
          }

          path = "/app1/*"
        }

        path {
          backend {
            service {
              name = "myapp-2"
              port {
                number = 8080
              }
            }
          }

          path = "/app2/*"
        }
      }
    }

    tls {
      secret_name = "tls-secret"
    }
  }
}

resource "kubernetes_service_v1" "example" {
  metadata {
    name = "myapp-1"
  }
  spec {
    selector = {
      app = kubernetes_pod_v1.example.metadata.0.labels.app
    }
    session_affinity = "ClientIP"
    port {
      port        = 8080
      target_port = 80
    }

    type = "NodePort"
  }
}

resource "kubernetes_service_v1" "example2" {
  metadata {
    name = "myapp-2"
  }
  spec {
    selector = {
      app = kubernetes_pod_v1.example2.metadata.0.labels.app
    }
    session_affinity = "ClientIP"
    port {
      port        = 8080
      target_port = 80
    }

    type = "NodePort"
  }
}

resource "kubernetes_pod_v1" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      app = "myapp-1"
    }
  }

  spec {
    container {
      image = "nginx:1.21.6"
      name  = "example"

      port {
        container_port = 80
      }
    }
  }
}

resource "kubernetes_pod_v1" "example2" {
  metadata {
    name = "terraform-example2"
    labels = {
      app = "myapp-2"
    }
  }

  spec {
    container {
      image = "nginx:1.21.6"
      name  = "example"

      port {
        container_port = 80
      }
    }
  }
}
```

## Example using Nginx ingress controller

```terraform
resource "kubernetes_service_v1" "example" {
  metadata {
    name = "ingress-service"
  }
  spec {
    port {
      port        = 80
      target_port = 80
      protocol    = "TCP"
    }
    type = "NodePort"
  }
}

resource "kubernetes_ingress_v1" "example" {
  wait_for_load_balancer = true
  metadata {
    name = "example"
  }
  spec {
    ingress_class_name = "nginx"
    rule {
      http {
        path {
          path = "/*"
          backend {
            service {
              name = kubernetes_service_v1.example.metadata.0.name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }
}

# Display load balancer hostname (typically present in AWS)
output "load_balancer_hostname" {
  value = kubernetes_ingress_v1.example.status.0.load_balancer.0.ingress.0.hostname
}

# Display load balancer IP (typically present in GCP, or using Nginx ingress controller)
output "load_balancer_ip" {
  value = kubernetes_ingress_v1.example.status.0.load_balancer.0.ingress.0.ip
}
```

## Import

Ingress can be imported using its namespace and name:

```
terraform import kubernetes_ingress_v1.<TERRAFORM_RESOURCE_NAME> <KUBE_NAMESPACE>/<KUBE_INGRESS_NAME>
```

e.g.

```
$ terraform import kubernetes_ingress_v1.example default/terraform-name
```
