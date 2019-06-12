resource "kubernetes_ingress" "example" {
  metadata {
    name = "example"

    annotations = {
      "ingress.kubernetes.io/rewrite-target" = "/"
    }
  }

  spec {
    backend {
      service_name = "echoserver"
      service_port = 8080
    }

    rule {
      host = "myminikube.info"

      http {
        path {
          path = "/"

          backend {
            service_name = "echoserver"
            service_port = 8080
          }
        }
      }
    }

    rule {
      host = "cheeses.all"

      http {
        path {
          path = "/stilton"

          backend {
            service_name = "stilton"
            service_port = 80
          }
        }

        path {
          path = "/cheddar"

          backend {
            service_name = "cheddar"
            service_port = 80
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "echoserver" {
  metadata {
    name = "echoserver"
  }

  spec {
    selector = {
      app = "echoserver"
    }

    port {
      port        = 8080
      target_port = 8080
    }

    type = "NodePort"
  }
}

resource "kubernetes_deployment" "echoserver" {
  metadata {
    name = "echoserver"
  }

  spec {
    selector {
      match_labels = {
        app = "echoserver"
      }
    }

    template {
      metadata {
        labels = {
          app = "echoserver"
        }
      }

      spec {
        container {
          name  = "echoserver"
          image = "gcr.io/google_containers/echoserver:1.4"

          port {
            container_port = 8080
          }
        }
      }
    }
  }
}

resource "kubernetes_deployment" "cheddar" {
  metadata {
    name = "cheddar-cheese"
  }

  spec {
    selector {
      match_labels = {
        app = "cheddar"
      }
    }

    template {
      metadata {
        labels = {
          app = "cheddar"
        }
      }

      spec {
        container {
          name  = "cheddar"
          image = "errm/cheese:cheddar"

          port {
            container_port = 80
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "cheddar" {
  metadata {
    name = "cheddar"
  }

  spec {
    selector = {
      app = "cheddar"
    }

    port {
      port        = 80
      target_port = 80
    }

    type = "NodePort"
  }
}

resource "kubernetes_deployment" "stilton" {
  metadata {
    name = "stilton-cheese"
  }

  spec {
    selector {
      match_labels = {
        app = "stilton"
      }
    }

    template {
      metadata {
        labels = {
          app = "stilton"
        }
      }

      spec {
        container {
          name  = "stilton"
          image = "errm/cheese:stilton"

          port {
            container_port = 80
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "stilton" {
  metadata {
    name = "stilton"
  }

  spec {
    selector = {
      app = "stilton"
    }

    port {
      port        = 80
      target_port = 80
    }

    type = "NodePort"
  }
}

output "ingress_ip" {
  value = kubernetes_ingress.example.load_balancer_ingress.*.ip
}
