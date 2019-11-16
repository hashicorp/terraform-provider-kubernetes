variable "minikube_host_ip" {}

variable "in_cluster_provider_version" {
  default = "1.10.0"
}
variable "in_cluster_provider_url" {
  default = ""
}

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
        version = "${var.in_cluster_provider_version}"

        load_config_file = false
      }

      resource "kubernetes_namespace" "test" {
        metadata {
          name = "test"
        }
      }
    EOT
  }
}

resource "kubernetes_service_account" "terraform" {
  metadata {
    name = "terraform"
  }

  automount_service_account_token = "true"
}

resource "kubernetes_cluster_role_binding" "terraform" {
  metadata {
    name = "terraform"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.terraform.metadata.0.name
    namespace = kubernetes_service_account.terraform.metadata.0.namespace
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
        service_account_name            = kubernetes_service_account.terraform.metadata.0.name
        automount_service_account_token = true

        container {
          name  = "terraform"
          image = "hashicorp/terraform:0.12.13"
          command = [
            "sh",
            "-c",
            "set && set -x && ${var.in_cluster_provider_url != "" ? "apk --no-cache add curl && mkdir -p ~/.terraform.d/plugins && curl ${var.in_cluster_provider_url} > ~/.terraform.d/plugins/terraform-provider-kubernetes_v${var.in_cluster_provider_version} && chmod +x ~/.terraform.d/plugins/* &&" : ""} mkdir /tf && cd /tf && cp /configuration/main.tf . && terraform init && TF_LOG=debug terraform plan && TF_LOG=debug terraform apply -auto-approve && sleep 10 && terraform destroy -auto-approve"
          ]

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
