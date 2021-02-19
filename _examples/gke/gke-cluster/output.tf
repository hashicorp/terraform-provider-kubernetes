output "node_version" {
  value = google_container_cluster.default.node_version
}

output "google_zone" {
  value = local.google_zone
}
