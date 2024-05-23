# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "test" {

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
