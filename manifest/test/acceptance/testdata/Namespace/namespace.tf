# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {

  manifest = {
    apiVersion = "v1"
    kind       = "Namespace"
    metadata = {
      name = var.name
    }
  }
}
