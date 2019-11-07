data "google_compute_zones" "available" {
}

data "google_container_engine_versions" "supported" {
  location       = "${data.google_compute_zones.available.names[0]}"
  version_prefix = "${var.kubernetes_version}"
}

resource "google_container_cluster" "primary" {
  name               = var.cluster_name
  location           = data.google_compute_zones.available.names[0]
  initial_node_count = 3

  node_version       = data.google_container_engine_versions.supported.latest_node_version
  min_master_version = data.google_container_engine_versions.supported.latest_master_version

  node_locations = [
    data.google_compute_zones.available.names[1],
  ]

  master_auth {
    username = var.username
    password = var.password
  }

  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}

output "cluster_name" {
  value = google_container_cluster.primary.name
}

output "primary_zone" {
  value = google_container_cluster.primary.zone
}

output "node_locations" {
  value = google_container_cluster.primary.node_locations
}

output "endpoint" {
  value = google_container_cluster.primary.endpoint
}

output "node_version" {
  value = google_container_cluster.primary.node_version
}

