resource "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "test"
  }

  endpoint {
    condition {
      ready = true
    }
    addresses = ["129.144.50.56"]
  }

  port {
    port = "9000"
    name = "first"
  }

  address_type = "IPv4"
}
