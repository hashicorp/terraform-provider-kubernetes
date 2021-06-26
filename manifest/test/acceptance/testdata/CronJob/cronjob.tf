provider "kubernetes-alpha" {
}

resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "batch/v1beta1"
    kind       = "CronJob"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      schedule = "0 * * * *"
      jobTemplate = {
        spec = {
          template = {
            metadata = {}
            spec = {
              restartPolicy = "Never"
              containers = [
                {
                  image = "busybox"
                  name  = "busybox"
                  command = [
                    "sleep", 
                    "30"
                  ]
                }
              ]
            }
          }
        }
      }
    }
  }
}
