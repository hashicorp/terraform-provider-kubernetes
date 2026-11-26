data "kubernetes_gateway_class_v1" "example" {
  metadata {
    name = "example"
  }
}

output "gateway_class_accepted" {
  description = "Whether the controller accepted the current generation of the GatewayClass."
  value = anytrue([
    for condition in data.kubernetes_gateway_class_v1.example.status.conditions : (
      condition.type == "Accepted" &&
      condition.status == "True" &&
      condition.observed_generation == data.kubernetes_gateway_class_v1.example.metadata[0].generation
    )
  ])
}
