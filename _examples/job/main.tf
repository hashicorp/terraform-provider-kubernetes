provider "kubernetes" {
}

resource "kubernetes_job" "test-pr" {
  metadata {
    name = "job-with-wait"
    namespace = "default"
  }
  spec {
    completions = 1
    template {
      metadata {}
      spec {
        container {
          name = "sleep"
          image = "busybox:latest"
          command = ["sleep", "30"]
        }
        restart_policy = "Never"
      }
    }
  }
  wait_for_completion = true
  timeouts {
    create = "40s"
  }
}