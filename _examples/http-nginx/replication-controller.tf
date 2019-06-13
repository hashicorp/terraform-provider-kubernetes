resource "kubernetes_replication_controller" "example" {
  metadata {
    name = "terraform-nginx-example"
    labels = {
      App = "TerraformNginxExample"
    }
  }

  spec {
    selector = {
      App = "TerraformNginxExample"
    }
    template {
      container {
        image = "nginx:${var.nginx_version}"
        name  = "example"

        port {
          container_port = 80
        }

        liveness_probe {
          http_get {
            path = "/index.html"
            port = 80
          }
          initial_delay_seconds = 30
          timeout_seconds       = 1
        }

        resources {
          limits {
            cpu    = "0.5"
            memory = "512Mi"
          }
          requests {
            cpu    = "250m"
            memory = "50Mi"
          }
        }
      }
    }
  }
}

