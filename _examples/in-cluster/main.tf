variable "minikube_host_ip" {}
variable "minikube_target_ip" {}

provider "kubernetes" {
  version = "1.10.0"

  host                   = "https://${var.minikube_host_ip}:8443"
  client_certificate     = file("~/.minikube/client.crt")
  client_key             = file("~/.minikube/client.key")
  cluster_ca_certificate = file("~/.minikube/ca.crt")
}

resource "kubernetes_config_map" "terraform" {
  metadata {
    name = "terraform"
  }

  data = {
    "main.tf" = <<-EOT
      provider "kubernetes" {
        version = "1.10"
      }

      resource "kubernetes_namespace" "test" {
        metadata {
          name = "test"
        }
      }
    EOT
  }
}

resource "kubernetes_job" "terraform" {
  metadata {
    name = "terraform"
  }
  spec {
    backoff_limit = 1
    template {
      metadata {}
      spec {
        container {
          name    = "terraform"
          image   = "hashicorp/terraform:0.12.13"
          command = ["sh", "-c", "mkdir /tf && cd /tf && cp /configuration/* . && KUBERNETES_SERVICE_HOST= && terraform init && terraform plan && terraform apply -auto-approve && sleep 10 && terraform destroy -auto-approve"]

          env {
            name  = "KUBE_HOST"
            value = "https://${var.minikube_target_ip}:8443"
          }
          env {
            name  = "KUBE_CLIENT_CERT_DATA"
            value = file("~/.minikube/client.crt")
          }
          env {
            name  = "KUBE_CLIENT_KEY_DATA"
            value = file("~/.minikube/client.key")
          }
          env {
            name  = "KUBE_CLUSTER_CA_CERT_DATA"
            value = file("~/.minikube/ca.crt")
          }

          volume_mount {
            name       = "configuration"
            mount_path = "/configuration"
          }
        }

        restart_policy = "Never"

        volume {
          name = "configuration"
          config_map {
            name = kubernetes_config_map.terraform.metadata[0].name
          }
        }
      }
    }
  }
}
