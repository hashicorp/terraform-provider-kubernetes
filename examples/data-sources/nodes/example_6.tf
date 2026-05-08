data "kubernetes_nodes" "example" {
  metadata {
    labels = {
      "kubernetes.io/os" = "linux"
    }
  }
}

output "linux-node-names" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.metadata.0.name]
}
