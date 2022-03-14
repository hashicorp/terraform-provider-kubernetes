
resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "autoscaling/v2beta2"
    kind       = "HorizontalPodAutoscaler"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      scaleTargetRef = {
        apiVersion = "apps/v1"
        kind       = "Deployment"
        name       = "nginx"
      }

      maxReplicas = 20
      minReplicas = 1

      metrics = [
        {
          type = "Resource"
          resource = {
            name = "cpu"
            target = {
              type               = "Utilization"
              averageUtilization = 65
            }
          }
        }
      ]
    }
  }
}
