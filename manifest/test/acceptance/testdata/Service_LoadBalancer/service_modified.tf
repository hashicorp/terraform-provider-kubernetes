# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name      = var.name
      namespace = var.namespace
      annotations = {
        test = "1"
      }
      labels = {
        test = "2"
      }
    }
    spec = {
      ports = [{
        name       = "https",
        port       = 443,
        targetPort = 8443,
        # Protcol is required for serverside apply per https://github.com/kubernetes-sigs/structured-merge-diff/issues/130
        protocol = "TCP"
      }]
      selector = {
        app = "test"
      }
      type = "LoadBalancer"
    }
  }
}
