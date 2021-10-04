resource "kubernetes_secret" "tfc-api-token" {
  metadata {
    name      = "terraformrc"
    namespace = kubernetes_manifest.namespace.object.metadata.name
    labels = {
      app = kubernetes_manifest.namespace.object.metadata.name
    }
  }

  data = {
    credentials = var.tfc_credentials
  }
}
