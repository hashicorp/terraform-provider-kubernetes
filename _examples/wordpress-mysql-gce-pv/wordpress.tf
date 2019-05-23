variable "wordpress_version" {
  default = "4.7.3"
}

resource "kubernetes_service" "wordpress" {
  metadata {
    name = "wordpress"
    labels = {
      app = "wordpress"
    }
  }
  spec {
    port {
      port        = 80
      target_port = 80
    }
    selector = {
      app  = "wordpress"
      tier = kubernetes_replication_controller.wordpress.spec[0].selector.tier
    }
    type = "LoadBalancer"
  }
}

output "lb_ip" {
  value = kubernetes_service.wordpress.load_balancer_ingress[0].ip
}

resource "kubernetes_persistent_volume_claim" "wordpress" {
  metadata {
    name = "wp-pv-claim"
    labels = {
      app = "wordpress"
    }
  }
  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "20Gi"
      }
    }
    volume_name = kubernetes_persistent_volume.wordpress.metadata[0].name
  }
}

resource "kubernetes_replication_controller" "wordpress" {
  metadata {
    name = "wordpress"
    labels = {
      app = "wordpress"
    }
  }
  spec {
    selector = {
      app  = "wordpress"
      tier = "frontend"
    }
    template {
      container {
        image = "wordpress:${var.wordpress_version}-apache"
        name  = "wordpress"

        env {
          name  = "WORDPRESS_DB_HOST"
          value = "wordpress-mysql"
        }
        env {
          name = "WORDPRESS_DB_PASSWORD"
          value_from {
            secret_key_ref {
              name = kubernetes_secret.mysql.metadata[0].name
              key  = "password"
            }
          }
        }

        port {
          container_port = 80
          name           = "wordpress"
        }

        volume_mount {
          name       = "wordpress-persistent-storage"
          mount_path = "/var/www/html"
        }
      }

      volume {
        name = "wordpress-persistent-storage"
        persistent_volume_claim {
          claim_name = kubernetes_persistent_volume_claim.wordpress.metadata[0].name
        }
      }
    }
  }
}

