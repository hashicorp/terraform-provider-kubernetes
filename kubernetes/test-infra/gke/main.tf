provider "google" {
  // Provider settings to be provided via ENV variables
}

data "google_compute_zones" "available" {}

resource "random_id" "cluster_name" {
  byte_length = 10
}

resource "random_id" "username" {
  byte_length = 14
}

resource "random_id" "password" {
  byte_length = 16
}

# See https://cloud.google.com/container-engine/supported-versions
variable "kubernetes_version" {}

resource "google_container_cluster" "primary" {
  name               = "tf-acc-test-${random_id.cluster_name.hex}"
  zone               = "${data.google_compute_zones.available.names[0]}"
  initial_node_count = 3
  node_version       = "${var.kubernetes_version}"
  min_master_version = "${var.kubernetes_version}"

  additional_zones = [
    "${data.google_compute_zones.available.names[1]}",
  ]

  master_auth {
    username = "${random_id.username.hex}"
    password = "${random_id.password.hex}"
  }

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

data "template_file" "kubeconfig" {
  template = "${file("${path.module}/kubeconfig-template.yaml")}"

  vars {
    cluster_name    = "${google_container_cluster.primary.name}"
    user_name       = "${google_container_cluster.primary.master_auth.0.username}"
    user_password   = "${google_container_cluster.primary.master_auth.0.password}"
    endpoint        = "${google_container_cluster.primary.endpoint}"
    cluster_ca      = "${google_container_cluster.primary.master_auth.0.cluster_ca_certificate}"
    client_cert     = "${google_container_cluster.primary.master_auth.0.client_certificate}"
    client_cert_key = "${google_container_cluster.primary.master_auth.0.client_key}"
  }
}

resource "local_file" "kubeconfig" {
  content  = "${data.template_file.kubeconfig.rendered}"
  filename = "${path.module}/kubeconfig"
}

output "google_zone" {
  value = "${data.google_compute_zones.available.names[0]}"
}

output "node_version" {
  value = "${google_container_cluster.primary.node_version}"
}

output "kubeconfig_path" {
  value = "${local_file.kubeconfig.filename}"
}
