# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      ports = [{
        name       = "http",
        port       = 80,
        targetPort = 8080,
        # Protcol is required for serverside apply per https://github.com/kubernetes-sigs/structured-merge-diff/issues/130
        protocol = "TCP"
      }]
      selector = {
        app = "test"
      }
      type = "NodePort"
    }
  }
}
