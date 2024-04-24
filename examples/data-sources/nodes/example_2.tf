# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_nodes" "example" {
  metadata {
    labels = {
      "kubernetes.io/os" = "linux"
    }
  }
}

output "linux-node-names" {
  value = [for node in data.kubernetes_nodes.example.nodes : node.metadata.0.name]
}
