resource "kubernetes_manifest" "test" {
  manifest = {
    // ...
  }

  wait {
    condition {
      type   = "ContainersReady"
      status = "True"
    }
  }
}
