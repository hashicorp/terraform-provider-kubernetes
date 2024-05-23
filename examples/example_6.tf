resource "kubernetes_deployment_v1" "this" {
  // omit the resource config
  lifecycle {
    ignore_changes = [
      spec[0].template[0].metadata[0].annotations["kubectl.kubernetes.io/restartedAt"],
    ]
  }
}
