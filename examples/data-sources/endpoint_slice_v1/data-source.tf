data "kubernetes_endpoint_slice_v1" "example" {
  metadata {
    name      = "example-endpoint-slice"
    namespace = "default"
  }
}

output "endpoint_addresses" {
  value = flatten([
    for e in data.kubernetes_endpoint_slice_v1.example.endpoint :
    e.addresses if e.condition[0].ready
  ])
}
