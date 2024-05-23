# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "node_version" {
  value = google_container_cluster.default.node_version
}

output "google_zone" {
  value = local.google_zone
}
