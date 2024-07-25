data "kubernetes_namespace" "example" {
  metadata {
    name = "kube-system"
  }
}
