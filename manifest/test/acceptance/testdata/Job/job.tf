provider "kubernetes-alpha" {
}

resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "batch/v1"
    kind       = "Job"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
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
