# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "service-account-injector" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ServiceAccount"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
      }
      "name"      = "${var.name}-vault-agent-injector"
      "namespace" = var.namespace
    }
  }
}
resource "kubernetes_manifest" "cluster-role-injector" {

  manifest = {
    "apiVersion" = "rbac.authorization.k8s.io/v1"
    "kind"       = "ClusterRole"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
      }
      "name" = "${var.name}-vault-agent-injector-clusterrole"
    }
    "rules" = [
      {
        "apiGroups" = [
          "admissionregistration.k8s.io",
        ]
        "resources" = [
          "mutatingwebhookconfigurations",
        ]
        "verbs" = [
          "get",
          "list",
          "watch",
          "patch",
        ]
      },
    ]
  }
}
resource "kubernetes_manifest" "cluster-role-binding-injector" {

  manifest = {
    "apiVersion" = "rbac.authorization.k8s.io/v1"
    "kind"       = "ClusterRoleBinding"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
      }
      "name" = "${var.name}-vault-agent-injector-binding"
    }
    "roleRef" = {
      "apiGroup" = "rbac.authorization.k8s.io"
      "kind"     = "ClusterRole"
      "name"     = "${var.name}-vault-agent-injector-clusterrole"
    }
    "subjects" = [
      {
        "kind"      = "ServiceAccount"
        "name"      = "${var.name}-vault-agent-injector"
        "namespace" = var.namespace
      },
    ]
  }
}
resource "kubernetes_manifest" "service-injector" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Service"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
      }
      "name"      = "${var.name}-vault-agent-injector-svc"
      "namespace" = var.namespace
    }
    "spec" = {
      "ports" = [
        {
          "port"       = 443
          "targetPort" = 8080
          "protocol"   = "TCP"
        },
      ]
      "selector" = {
        "app.kubernetes.io/instance" = var.name
        "app.kubernetes.io/name"     = "vault-agent-injector"
        "component"                  = "webhook"
      }
    }
  }
}
resource "kubernetes_manifest" "deployment-injector" {

  manifest = {
    "apiVersion" = "apps/v1"
    "kind"       = "Deployment"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
        "component"                    = "webhook"
      }
      "name"      = "${var.name}-vault-agent-injector"
      "namespace" = var.namespace
    }
    "spec" = {
      "replicas" = 1
      "selector" = {
        "matchLabels" = {
          "app.kubernetes.io/instance" = var.name
          "app.kubernetes.io/name"     = "vault-agent-injector"
          "component"                  = "webhook"
        }
      }
      "template" = {
        "metadata" = {
          "labels" = {
            "app.kubernetes.io/instance" = var.name
            "app.kubernetes.io/name"     = "vault-agent-injector"
            "component"                  = "webhook"
          }
        }
        "spec" = {
          "containers" = [
            {
              "args" = [
                "agent-inject",
                "2>&1",
              ]
              "env" = [
                {
                  "name"  = "AGENT_INJECT_LISTEN"
                  "value" = ":8080"
                },
                {
                  "name"  = "AGENT_INJECT_LOG_LEVEL"
                  "value" = "info"
                },
                {
                  "name"  = "AGENT_INJECT_VAULT_ADDR"
                  "value" = "http://${var.name}-vault.default.svc:${var.server_service.port}"
                },
                {
                  "name"  = "AGENT_INJECT_VAULT_AUTH_PATH"
                  "value" = "auth/kubernetes"
                },
                {
                  "name"  = "AGENT_INJECT_VAULT_IMAGE"
                  "value" = var.vault_image
                },
                {
                  "name"  = "AGENT_INJECT_TLS_AUTO"
                  "value" = "${var.name}-vault-agent-injector-cfg"
                },
                {
                  "name"  = "AGENT_INJECT_TLS_AUTO_HOSTS"
                  "value" = "${var.name}-vault-agent-injector-svc,${var.name}-vault-agent-injector-svc.default,${var.name}-vault-agent-injector-svc.default.svc"
                },
                {
                  "name"  = "AGENT_INJECT_LOG_FORMAT"
                  "value" = "standard"
                },
                {
                  "name"  = "AGENT_INJECT_REVOKE_ON_SHUTDOWN"
                  "value" = "false"
                },
              ]
              "image"           = var.vault_k8s_image
              "imagePullPolicy" = "IfNotPresent"
              "livenessProbe" = {
                "failureThreshold" = 2
                "httpGet" = {
                  "path"   = "/health/ready"
                  "port"   = 8080
                  "scheme" = "HTTPS"
                }
                "initialDelaySeconds" = 1
                "periodSeconds"       = 2
                "successThreshold"    = 1
                "timeoutSeconds"      = 5
              }
              "name" = "sidecar-injector"
              "readinessProbe" = {
                "failureThreshold" = 2
                "httpGet" = {
                  "path"   = "/health/ready"
                  "port"   = 8080
                  "scheme" = "HTTPS"
                }
                "initialDelaySeconds" = 2
                "periodSeconds"       = 2
                "successThreshold"    = 1
                "timeoutSeconds"      = 5
              }
            },
          ]
          "securityContext" = {
            "runAsGroup"   = 1000
            "runAsNonRoot" = true
            "runAsUser"    = 100
          }
          "serviceAccountName" = "${var.name}-vault-agent-injector"
        }
      }
    }
  }
}
