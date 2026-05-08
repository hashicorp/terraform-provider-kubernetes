resource "kubernetes_manifest" "test" {
  manifest = {
    // ...
  }

  wait {
    rollout = true
  }
}
