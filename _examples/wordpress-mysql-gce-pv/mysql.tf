variable "mysql_password" {
}

variable "mysql_version" {
  default = "5.6"
}

resource "kubernetes_service" "mysql" {
  metadata {
    name = "wordpress-mysql"
    labels = {
      app = "wordpress"
    }
  }
  spec {
    port {
      port        = 3306
      target_port = 3306
    }
    selector = {
      app  = "wordpress"
      tier = kubernetes_replication_controller.mysql.spec[0].selector.tier
    }
    cluster_ip = "None"
  }
}

resource "kubernetes_persistent_volume_claim" "mysql" {
  metadata {
    name = "mysql-pv-claim"
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
    volume_name = kubernetes_persistent_volume.mysql.metadata[0].name
  }
}

resource "kubernetes_secret" "mysql" {
  metadata {
    name = "mysql-pass"
  }

  data = {
    password = var.mysql_password
  }
}

resource "kubernetes_replication_controller" "mysql" {
  metadata {
    name = "wordpress-mysql"
    labels = {
      app = "wordpress"
    }
  }
  spec {
    selector = {
      app  = "wordpress"
      tier = "mysql"
    }
    template {
      container {
        image = "mysql:${var.mysql_version}"
        name  = "mysql"

        env {
          name = "MYSQL_ROOT_PASSWORD"
          value_from {
            secret_key_ref {
              name = kubernetes_secret.mysql.metadata[0].name
              key  = "password"
            }
          }
        }

        port {
          container_port = 3306
          name           = "mysql"
        }

        volume_mount {
          name       = "mysql-persistent-storage"
          mount_path = "/var/lib/mysql"
        }
      }

      volume {
        name = "mysql-persistent-storage"
        persistent_volume_claim {
          claim_name = kubernetes_persistent_volume_claim.mysql.metadata[0].name
        }
      }
    }
  }
}

