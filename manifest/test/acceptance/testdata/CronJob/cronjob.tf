# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "batch/v1"
    kind       = "CronJob"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      schedule = "0 * * * *"
      jobTemplate = {
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
  }
}
