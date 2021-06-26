resource "kubernetes_secret" "workspace-secret" {
  metadata {
    name      = var.workspace_secrets
    namespace = kubernetes_manifest.namespace.object.metadata.name
  }

  data = {
    access_key_id    = var.access_key_id
    secret_acess_key = var.secret_acess_key
  }
}
