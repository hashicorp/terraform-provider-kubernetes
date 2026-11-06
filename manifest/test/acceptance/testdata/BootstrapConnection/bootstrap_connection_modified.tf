# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

# bootstrap_connection.tf with "config_path" swapped for an equivalent
# "config_paths". See TestKubernetesManifest_BootstrapConnection.
resource "terraform_data" "bootstrap" {
  input = var.kubeconfig
}

resource "kubernetes_manifest" "test" {
  bootstrap_connection {
    config_paths = [terraform_data.bootstrap.output]
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
