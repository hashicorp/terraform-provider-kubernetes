# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "kubeconfig_path" {
  value = abspath("${path.root}/kubeconfig")
}

output "cluster_name" {
  value = local.cluster_name
}

output "google_zone" {
  value = module.gke-cluster.google_zone
}
