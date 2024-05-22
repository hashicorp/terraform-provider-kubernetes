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
