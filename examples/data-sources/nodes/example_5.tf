data "kubernetes_nodes" "example" {}

output "node-ids" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.spec.0.provider_id]
}
