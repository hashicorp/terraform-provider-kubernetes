# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_job" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    template {
      metadata {}
      spec {
        container {
          name    = "pi"
          image   = "alpine"
          command = ["sh", "-c", "sleep 10"]
        }
        restart_policy = "Never"
      }
    }
    backoff_limit = 4
  }
  wait_for_completion = false
}
