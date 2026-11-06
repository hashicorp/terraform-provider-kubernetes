# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

# terraform_data stands in for a cluster created earlier in the same apply: its "output"
# is not known until apply, so the "bootstrap_connection" block is used.
resource "terraform_data" "bootstrap" {
  input = var.kubeconfig
}

resource "kubernetes_manifest" "test" {
  bootstrap_connection {
    config_path = terraform_data.bootstrap.output
  }

  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    data = {
      foo = "bar"
    }
  }
}
