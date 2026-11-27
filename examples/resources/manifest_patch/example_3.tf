# A setting in a ConfigMap that a Helm chart created.
resource "kubernetes_manifest_patch" "argocd_cm" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = "argocd-cm"
      namespace = "argocd"
    }
    data = {
      "timeout.reconciliation" = "300s"
    }
  }

  depends_on = [helm_release.argocd]
}

# A label of a namespace that the cluster created.
resource "kubernetes_manifest_patch" "default_namespace" {
  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name   = "default"
      labels = { "istio-injection" = "enabled" }
    }
  }
}
