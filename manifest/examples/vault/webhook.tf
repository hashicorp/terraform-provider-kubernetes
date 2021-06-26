resource "kubernetes_manifest" "webhook-injector" {
  provider = kubernetes-alpha
  manifest = {
    "apiVersion" = "admissionregistration.k8s.io/v1beta1"
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
