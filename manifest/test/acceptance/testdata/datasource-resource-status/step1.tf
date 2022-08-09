resource "kubernetes_deployment" "test_deploy" {
  api_version = "v1"

  metadata = {
    name = var.name
    namespace = var.namespace
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        test = "test-status"
      }
    }

    template {
      metadata {
        labels = {
          test = "test-status"
        }
      }

      spec {
        container {
          image = "nginx:1.21.6"
          name  = "test"
          }
        }
      }
    }
  }
}
