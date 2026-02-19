# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_deployment" "tfc-deployment" {
  metadata {
    name      = "${kubernetes_manifest.namespace.object.metadata.name}-sync-workspace"
    namespace = kubernetes_manifest.namespace.object.metadata.name
    labels = {
      app = kubernetes_manifest.namespace.object.metadata.name
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app       = kubernetes_manifest.namespace.object.metadata.name
        component = "sync-workspace"
      }
    }
    template {
      metadata {
        labels = {
          app       = kubernetes_manifest.namespace.object.metadata.name
          component = "sync-workspace"
        }
      }

      spec {
        service_account_name            = kubernetes_service_account.tfc-service-account.metadata[0].name
        automount_service_account_token = true

        volume {
          name = "terraformrc"
          secret {
            secret_name = var.terraformrc_secret_name
            items {
              key  = var.terraformrc_secret_key
              path = ".terraformrc"
            }
          }
        }

        volume {
          name = "sensitivevars"

          secret {
            secret_name = var.workspace_secrets
          }
        }

        container {
          name    = "terraform-sync-workspace"
          image   = var.sync_workspace_image
          command = ["/bin/sh", "-ec", "terraform-k8s sync-workspace \\\n  --k8s-watch-namespace=\"${kubernetes_manifest.namespace.object.metadata.name}\""]

          env {
            name = "POD_NAME"

            value_from {
              field_ref {
                field_path = "metadata.name"
              }
            }
          }

          env {
            name  = "OPERATOR_NAME"
            value = "terraform-k8s"
          }

          env {
            name  = "TF_CLI_CONFIG_FILE"
            value = "/etc/terraform/.terraformrc"
          }

          volume_mount {
            name       = "terraformrc"
            read_only  = true
            mount_path = "/etc/terraform"
          }

          volume_mount {
            name       = "sensitivevars"
            read_only  = true
            mount_path = "/tmp/secrets"
          }

          liveness_probe {
            http_get {
              path   = "/metrics"
              port   = "8383"
              scheme = "HTTP"
            }

            initial_delay_seconds = 30
            timeout_seconds       = 5
            period_seconds        = 5
            success_threshold     = 1
            failure_threshold     = 3
          }

          readiness_probe {
            http_get {
              path   = "/metrics"
              port   = "8383"
              scheme = "HTTP"
            }

            initial_delay_seconds = 10
            timeout_seconds       = 5
            period_seconds        = 5
            success_threshold     = 1
            failure_threshold     = 5
          }
        }
      }
    }
  }
}
