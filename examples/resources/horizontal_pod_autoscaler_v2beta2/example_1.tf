resource "kubernetes_horizontal_pod_autoscaler_v2beta2" "example" {
  metadata {
    name = "test"
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }

    metric {
      type = "External"
      external {
        metric {
          name = "latency"
          selector {
            match_labels = {
              lb_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "100"
        }
      }
    }
  }
}
