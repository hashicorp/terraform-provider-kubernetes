resource "kubernetes_gateway_class_v1" "example" {
  metadata {
    name = "example"
  }

  spec {
    controller_name = "example.com/gateway-controller"
    description     = "Example GatewayClass"
  }
}
