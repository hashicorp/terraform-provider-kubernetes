data "kubernetes_resource" "example" {
  api_version = "v1"
  kind        = "ConfigMap"

  metadata {
    name      = "example"
    namespace = "default"
  }
}

output "test" {
  value = data.kubernetes_resource.example.object.data.TEST
}
