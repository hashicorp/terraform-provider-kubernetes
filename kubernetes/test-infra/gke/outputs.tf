# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

output "google_zone" {
  value = data.google_compute_zones.available.names[0]
}

output "node_version" {
  value = google_container_cluster.primary.node_version
}

output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}

output "cluster_name" {
  value = google_container_cluster.primary.name
}
