
resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = var.group_version
    kind       = var.kind
    metadata = {
      namespace = var.namespace
      name      = var.name
    }
    data = {
      nested = {
        testdata = var.testdata
      }
    }
  }
}
