# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test-ns" {
  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name = var.namespace
    }
  }
}

resource "tls_private_key" "ca_key" {
  algorithm = "RSA"
}

resource "tls_self_signed_cert" "ca_cert" {
  private_key_pem = tls_private_key.ca_key.private_key_pem

  is_ca_certificate = true

  subject {
    common_name  = var.name
    organization = "Hashicorp"
  }

  validity_period_hours = 12

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "cert_signing",
  ]
}

resource "tls_cert_request" "svc_cert_req" {
  private_key_pem = tls_private_key.ca_key.private_key_pem

  subject {
    common_name  = "${kubernetes_manifest.service_annotate_webhook.manifest.metadata.name}.${kubernetes_manifest.test-ns.object.metadata.name}.svc"
    organization = "Hashicorp"
  }

  dns_names = [
    "${kubernetes_manifest.service_annotate_webhook.manifest.metadata.name}.${kubernetes_manifest.test-ns.object.metadata.name}.svc"
  ]
}

resource "tls_locally_signed_cert" "svc_cert" {
  cert_request_pem   = tls_cert_request.svc_cert_req.cert_request_pem
  ca_private_key_pem = tls_private_key.ca_key.private_key_pem
  ca_cert_pem        = tls_self_signed_cert.ca_cert.cert_pem

  validity_period_hours = 12

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]
}

resource "kubernetes_manifest" "secret_annotate_webhook_certs" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Secret"
    "metadata" = {
      "name"      = var.name
      "namespace" = kubernetes_manifest.test-ns.object.metadata.name
    }
    "data" = {
      "cert.pem" = base64encode(tls_locally_signed_cert.svc_cert.cert_pem)
      "key.pem"  = base64encode(tls_private_key.ca_key.private_key_pem)
    }
  }
}

resource "kubernetes_manifest" "deployment_annotate_webhook" {
  manifest = {
    "apiVersion" = "apps/v1"
    "kind"       = "Deployment"
    "metadata" = {
      "labels" = {
        "app" = var.name
      }
      "name"      = var.name
      "namespace" = kubernetes_manifest.test-ns.object.metadata.name
    }
    "spec" = {
      "replicas" = 1
      "selector" = {
        "matchLabels" = {
          "app" = var.name
        }
      }
      "template" = {
        "metadata" = {
          "labels" = {
            "app" = var.name
          }
        }
        "spec" = {
          "containers" = [
            {
              "image"           = var.webhook_image
              "imagePullPolicy" = "Never"
              "name"            = var.name
              "volumeMounts" = [
                {
                  "mountPath" = "/etc/webhook/certs"
                  "name"      = "webhook-certs"
                  "readOnly"  = true
                },
              ]
            },
          ]
          "volumes" = [
            {
              "name" = "webhook-certs"
              "secret" = {
                "secretName" = kubernetes_manifest.secret_annotate_webhook_certs.object.metadata.name
              }
            },
          ]
        }
      }
    }
  }
}

resource "kubernetes_manifest" "service_annotate_webhook" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Service"
    "metadata" = {
      "labels" = {
        "app" = var.name
      }
      "name"      = var.name
      "namespace" = kubernetes_manifest.test-ns.object.metadata.name
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
        "app" = var.name
      }
    }
  }
}

resource "kubernetes_manifest" "mutatingwebhookconfiguration_annotate_webhook" {
  manifest = {
    "apiVersion" = "admissionregistration.k8s.io/v1"
    "kind"       = "MutatingWebhookConfiguration"
    "metadata" = {
      "labels" = {
        "app"  = var.name
        "kind" = "mutator"
      }
      "name" = var.name
    }
    "webhooks" = [
      {
        "admissionReviewVersions" = [
          "v1",
        ]
        "clientConfig" = {
          "caBundle" = base64encode(tls_self_signed_cert.ca_cert.cert_pem)
          "service" = {
            "name"      = var.name
            "namespace" = kubernetes_manifest.test-ns.object.metadata.name
            "path"      = "/mutate"
          }
        }
        "name" = "${var.name}.hashicorp.com"
        "rules" = [
          {
            "apiGroups" = [
              "",
            ]
            "apiVersions" = [
              "v1",
            ]
            "operations" = [
              "CREATE",
            ]
            "resources" = [
              "*",
            ]
          },
        ]
        "sideEffects" = "None"
      },
    ]
  }
  depends_on = [kubernetes_manifest.deployment_annotate_webhook]
}
