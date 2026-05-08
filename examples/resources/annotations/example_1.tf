resource "kubernetes_annotations" "example" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = "my-config"
  }
  annotations = {
    "owner" = "myteam"
  }
}
