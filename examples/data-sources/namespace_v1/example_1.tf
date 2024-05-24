data "kubernetes_namespace_v1" "example" {
  metadata {
    name = "kube-system"
  }
}
