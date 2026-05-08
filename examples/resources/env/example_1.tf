resource "kubernetes_env" "example" {
  container = "nginx"
  metadata {
    name = "nginx-deployment"
  }

  api_version = "apps/v1"
  kind        = "Deployment"

  env {
    name  = "NGINX_HOST"
    value = "google.com"
  }

  env {
    name  = "NGINX_PORT"
    value = "90"
  }
}
