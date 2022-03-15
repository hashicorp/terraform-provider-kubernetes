data "kubernetes_resource" "test" {
  api_version = "v1"
  kind        = "Pod"

  metadata {
    name      = var.name
    namespace = var.namespace
  }

  wait_for = {
    fields = {
      "status.containerStatuses[0].restartCount" = "0",
      "status.containerStatuses[0].ready"        = "true",

      "status.podIP" = "^(\\d+(\\.|$)){4}",
      "status.phase" = "Running",
    }
  }
}
