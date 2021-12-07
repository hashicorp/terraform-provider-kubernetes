variable "kubernetes_version" {
  default = ""
}

variable "workers_count" {
  default = "3"
}

data "google_compute_zones" "available" {
}

data "google_client_config" "default" {
}

data "google_container_engine_versions" "supported" {
  location       = data.google_compute_zones.available.names[0]
  version_prefix = var.kubernetes_version
}

resource "random_id" "cluster_name" {
  byte_length = 10
}

resource "google_service_account" "default" {
  account_id   = "tf-k8s-${random_id.cluster_name.hex}"
  display_name = "Kubernetes provider SA"
}

resource "google_container_cluster" "primary" {
  name               = "tf-acc-test-${random_id.cluster_name.hex}"
  location           = data.google_compute_zones.available.names[0]
  initial_node_count = var.workers_count
  node_version       = data.google_container_engine_versions.supported.latest_node_version
  min_master_version = data.google_container_engine_versions.supported.latest_master_version

  node_locations = [
    data.google_compute_zones.available.names[1],
  ]

  node_config {
    machine_type    = "n1-standard-4"
    service_account = google_service_account.default.email

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}

locals {
  kubeconfig = {
    apiVersion = "v1"
    kind       = "Config"
    preferences = {
      colors = true
    }
    current-context = "tf-k8s-gcp-test"
    contexts = [
      {
        name = "tf-k8s-gcp-test"
        context = {
          cluster   = google_container_cluster.primary.name
          user      = google_service_account.default.email
          namespace = "default"
        }
      }
    ]
    clusters = [
      {
        name = google_container_cluster.primary.name
        cluster = {
          server                     = "https://${google_container_cluster.primary.endpoint}"
          certificate-authority-data = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
        }
      }
    ]
    users = [
      {
        name = google_service_account.default.email
        user = {
          auth-provider = {
            name = "gcp"
            config = {
              cmd-args = "config config-helper --format=json --access-token-file=gcptoken"
              cmd-path = "gcloud"
              expiry-key = "{.credential.token_expiry}"
              token-key = "{.credential.access_token}"
            }
          }
        }
      }
    ]
  }
}

resource "local_file" "kubeconfig" {
  content  = yamlencode(local.kubeconfig)
  filename = "${path.module}/kubeconfig"
}

resource "local_file" "gcptoken" {
  content  = data.google_client_config.default.access_token
  filename = "${path.module}/gcptoken"
}

output "google_zone" {
  value = data.google_compute_zones.available.names[0]
}

output "node_version" {
  value = google_container_cluster.primary.node_version
}

output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}

