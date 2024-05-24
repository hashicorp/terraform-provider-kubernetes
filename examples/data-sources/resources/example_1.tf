data "kubernetes_resources" "example" {
  api_version    = "v1"
  kind           = "Namespace"
  field_selector = "metadata.name!=kube-system"
}

output "test" {
  value = length(data.kubernetes_resources.example.objects)
}
