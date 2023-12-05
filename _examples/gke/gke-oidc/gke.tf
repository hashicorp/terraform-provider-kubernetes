# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "google_container_cluster" "upstream" {
  provider = google-beta
  name     = var.cluster_name
  location = var.gke_location
}

data "google_client_config" "provider" {
  provider = google-beta
}
