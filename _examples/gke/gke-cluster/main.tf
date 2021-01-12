provider "google" {
  # Provider is configured using environment variables: GOOGLE_REGION, GOOGLE_PROJECT, GOOGLE_CREDENTIALS.
  # This can be set statically, if preferred. See docs for details.
  # https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#full-reference
}

# This is used to set local variable google_zone.
# This can be replaced with a statically-configured zone, if preferred.
data "google_compute_zones" "available" {
}

data "google_container_engine_versions" "supported" {
  location           = local.google_zone
  version_prefix     = var.kubernetes_version
}

resource "google_container_cluster" "primary" {
  name               = var.cluster_name
  location           = local.google_zone
  initial_node_count = var.workers_count
  min_master_version = data.google_container_engine_versions.supported.latest_master_version
  # node version must match master version
  # https://www.terraform.io/docs/providers/google/r/container_cluster.html#node_version
  node_version       = data.google_container_engine_versions.supported.latest_master_version

  node_locations = [
    data.google_compute_zones.available.names[1],
  ]

  node_config {
    machine_type = "n1-standard-4"

    oauth_scopes = [
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}
