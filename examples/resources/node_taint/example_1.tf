resource "kubernetes_node_taint" "example" {
  metadata {
    name = "my-node.my-cluster.k8s.local"
  }
  taint {
    key    = "node-role.kubernetes.io/example"
    value  = "true"
    effect = "NoSchedule"
  }
}
