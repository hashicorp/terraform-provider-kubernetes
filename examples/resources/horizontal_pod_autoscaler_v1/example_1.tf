resource "kubernetes_horizontal_pod_autoscaler_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  spec {
    max_replicas = 10
    min_replicas = 8

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }
  }
}
