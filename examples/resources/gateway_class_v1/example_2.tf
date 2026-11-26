resource "kubernetes_gateway_class_v1" "internal" {
  metadata {
    name = "internal"
  }

  spec {
    controller_name = "example.com/gateway-controller"
    description     = "Gateways for internal traffic"

    parameters_ref {
      group     = "example.com"
      kind      = "GatewayConfig"
      name      = "internal"
      namespace = "gateway-system"
    }
  }
}
