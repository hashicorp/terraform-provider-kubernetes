# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

locals {
  server_shell_script = <<EOT
sed -E "s/HOST_IP/$${HOST_IP?}/g" /vault/config/extraconfig-from-values.hcl > /tmp/storageconfig.hcl;
sed -Ei "s/POD_IP/$${POD_IP?}/g" /tmp/storageconfig.hcl;
/usr/local/bin/docker-entrypoint.sh vault server -config=/tmp/storageconfig.hcl
EOT
}
resource "kubernetes_manifest" "service-account-vault" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ServiceAccount"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault"
      }
      "name"      = "${var.name}-vault"
      "namespace" = var.namespace
    }
  }
}
resource "kubernetes_manifest" "cluster-role-binding-server" {

  manifest = {
    "apiVersion" = "rbac.authorization.k8s.io/v1"
    "kind"       = "ClusterRoleBinding"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault"
      }
      "name" = "${var.name}-vault-server-binding"
    }
    "roleRef" = {
      "apiGroup" = "rbac.authorization.k8s.io"
      "kind"     = "ClusterRole"
      "name"     = "system:auth-delegator"
    }
    "subjects" = [
      {
        "kind"      = "ServiceAccount"
        "name"      = "${var.name}-vault"
        "namespace" = var.namespace
      },
    ]
  }
}
resource "kubernetes_manifest" "service-server" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Service"
    "metadata" = {
      "annotations" = merge({
        "service.alpha.kubernetes.io/tolerate-unready-endpoints" = "true"
      }, var.server_service.annotations)
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault"
      }
      "name"      = "${var.name}-vault"
      "namespace" = var.namespace
    }
    "spec" = {
      "ports" = [
        {
          "name"       = "http"
          "port"       = var.server_service.port
          "targetPort" = var.server_service.targetPort
          "protocol"   = "TCP"
        },
        {
          "name"       = "https-internal"
          "port"       = 8201
          "targetPort" = 8201
          "protocol"   = "TCP"
        },
      ]
      "publishNotReadyAddresses" = true
      "selector" = {
        "app.kubernetes.io/instance" = var.name
        "app.kubernetes.io/name"     = "vault"
        "component"                  = "server"
      }
    }
  }
}
resource "kubernetes_manifest" "statefulset-server" {

  manifest = {
    "apiVersion" = "apps/v1"
    "kind"       = "StatefulSet"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault"
      }
      "name"      = "${var.name}-vault"
      "namespace" = var.namespace
    }
    "spec" = {
      "podManagementPolicy" = "Parallel"
      "replicas"            = 1
      "selector" = {
        "matchLabels" = {
          "app.kubernetes.io/instance" = var.name
          "app.kubernetes.io/name"     = "vault"
          "component"                  = "server"
        }
      }
      "serviceName" = "${var.name}-vault-internal"
      "template" = {
        "metadata" = {
          "labels" = {
            "app.kubernetes.io/instance" = var.name
            "app.kubernetes.io/name"     = "vault"
            "component"                  = "server"
          }
        }
        "spec" = {
          "affinity" = {
            "podAntiAffinity" = {
              "requiredDuringSchedulingIgnoredDuringExecution" = [
                {
                  "labelSelector" = {
                    "matchLabels" = {
                      "app.kubernetes.io/instance" = var.name
                      "app.kubernetes.io/name"     = "vault"
                      "component"                  = "server"
                    }
                  }
                  "topologyKey" = "kubernetes.io/hostname"
                },
              ]
            }
          }
          "containers" = [
            {
              "args" = [
                local.server_shell_script
              ]
              "command" = [
                "/bin/sh",
                "-ec",
              ]
              "env" = [
                {
                  "name" = "HOST_IP"
                  "valueFrom" = {
                    "fieldRef" = {
                      "fieldPath" = "status.hostIP"
                    }
                  }
                },
                {
                  "name" = "POD_IP"
                  "valueFrom" = {
                    "fieldRef" = {
                      "fieldPath" = "status.podIP"
                    }
                  }
                },
                {
                  "name" = "VAULT_K8S_POD_NAME"
                  "valueFrom" = {
                    "fieldRef" = {
                      "fieldPath" = "metadata.name"
                    }
                  }
                },
                {
                  "name" = "VAULT_K8S_NAMESPACE"
                  "valueFrom" = {
                    "fieldRef" = {
                      "fieldPath" = "metadata.namespace"
                    }
                  }
                },
                {
                  "name"  = "VAULT_ADDR"
                  "value" = "http://127.0.0.1:8200"
                },
                {
                  "name"  = "VAULT_API_ADDR"
                  "value" = "http://$(POD_IP):8200"
                },
                {
                  "name"  = "SKIP_CHOWN"
                  "value" = "true"
                },
                {
                  "name"  = "SKIP_SETCAP"
                  "value" = "true"
                },
                {
                  "name" = "HOSTNAME"
                  "valueFrom" = {
                    "fieldRef" = {
                      "fieldPath" = "metadata.name"
                    }
                  }
                },
                {
                  "name"  = "VAULT_CLUSTER_ADDR"
                  "value" = "https://$(HOSTNAME).${var.name}-vault-internal:8201"
                },
              ]
              "image"           = var.vault_image
              "imagePullPolicy" = "IfNotPresent"
              "lifecycle" = {
                "preStop" = {
                  "exec" = {
                    "command" = [
                      "/bin/sh",
                      "-c",
                      "sleep 5 && kill -SIGTERM $(pidof vault)",
                    ]
                  }
                }
              }
              "name" = "vault"
              "ports" = [
                {
                  "containerPort" = 8200
                  "name"          = "http"
                  "protocol"      = "TCP"
                },
                {
                  "containerPort" = 8201
                  "name"          = "https-internal"
                  "protocol"      = "TCP"
                },
                {
                  "containerPort" = 8202
                  "name"          = "http-rep"
                  "protocol"      = "TCP"
                },
              ]
              "readinessProbe" = {
                "exec" = {
                  "command" = [
                    "/bin/sh",
                    "-ec",
                    "vault status -tls-skip-verify",
                  ]
                }
                "failureThreshold"    = 2
                "initialDelaySeconds" = 5
                "periodSeconds"       = 3
                "successThreshold"    = 1
                "timeoutSeconds"      = 5
              }
              "volumeMounts" = [
                {
                  "mountPath" = "/vault/data"
                  "name"      = "data"
                },
                {
                  "mountPath" = "/vault/config"
                  "name"      = "config"
                },
              ]
            },
          ]
          "securityContext" = {
            "fsGroup"      = 1000
            "runAsGroup"   = 1000
            "runAsNonRoot" = true
            "runAsUser"    = 100
          }
          "serviceAccountName"            = "${var.name}-vault"
          "terminationGracePeriodSeconds" = 10
          "volumes" = [
            {
              "configMap" = {
                "name" = "${var.name}-vault-config"
              }
              "name" = "config"
            },
          ]
        }
      }
      "updateStrategy" = {
        "type" = "OnDelete"
      }
      "volumeClaimTemplates" = [
        {
          "metadata" = {
            "name" = "data"
          }
          "spec" = {
            "accessModes" = [
              "ReadWriteOnce",
            ]
            "resources" = {
              "requests" = {
                "storage" = "10Gi"
              }
            }
          }
        },
      ]
    }
  }
}
