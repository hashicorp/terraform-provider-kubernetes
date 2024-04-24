# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_persistent_volume" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    capacity = {
      storage = "2Gi"
    }
    access_modes = ["ReadWriteMany"]
    persistent_volume_source {
      vsphere_volume {
        volume_path = "/absolute/path"
      }
    }
  }
}
