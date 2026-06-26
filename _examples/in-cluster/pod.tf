# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_pod_v1" "this" {
  metadata {
    name      = "this"
    namespace = "default"
  }
  spec {
    container {
      name    = "this"
      image   = "busybox"
      command = ["sleep", "infinity"]
    }
  }
}
