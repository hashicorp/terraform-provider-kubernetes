# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

output "kubeconfig_path" {
  value = abspath("${path.root}/kubeconfig")
}

output "cluster_name" {
  value = local.cluster_name
}
