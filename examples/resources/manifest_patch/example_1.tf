# Set the replica count on a Deployment owned by another tool (e.g. Helm), without
# taking over the whole object. This resource owns only spec.replicas.
resource "kubernetes_manifest_patch" "scale" {
  api_version = "apps/v1"
  kind        = "Deployment"
  name        = "app"
  namespace   = "default"

  patch = {
    spec = {
      replicas = 3
    }
  }
}
