# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

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
