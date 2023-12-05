# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "webhook-injector" {

  manifest = {
    "apiVersion" = "admissionregistration.k8s.io/v1"
    "kind"       = "MutatingWebhookConfiguration"
    "metadata" = {
      "labels" = {
        "app.kubernetes.io/instance"   = var.name
        "app.kubernetes.io/managed-by" = "Terraform"
        "app.kubernetes.io/name"       = "vault-agent-injector"
      }
      "name" = "${var.name}-vault-agent-injector-cfg"
    }
    "webhooks" = [
      {
        "clientConfig" = {
          "service" = {
            "name"      = "${var.name}-vault-agent-injector-svc"
            "namespace" = var.namespace
            "path"      = "/mutate"
          }
        }
        "name" = "vault.hashicorp.com"
        "admissionReviewVersions" = [
          "v1",
        ]
        "sideEffects" = "None"
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
              "UPDATE",
            ]
            "resources" = [
              "pods",
            ]
          },
        ]
      },
    ]

  }
}
