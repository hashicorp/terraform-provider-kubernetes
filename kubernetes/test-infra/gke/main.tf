# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

data "google_compute_zones" "available" {
}

data "google_container_engine_versions" "supported" {
  location       = data.google_compute_zones.available.names[0]
  version_prefix = var.cluster_version
}

resource "random_id" "cluster_name" {
  byte_length = 10
}

resource "google_service_account" "default" {
  account_id   = "tf-k8s-${random_id.cluster_name.hex}"
  display_name = "Kubernetes provider SA"
}

resource "google_container_cluster" "primary" {
  provider           = google-beta
  name               = var.cluster_name != "" ? var.cluster_name : "tf-acc-test-${random_id.cluster_name.hex}"
  location           = data.google_compute_zones.available.names[0]
  node_version       = data.google_container_engine_versions.supported.latest_node_version
  min_master_version = data.google_container_engine_versions.supported.latest_master_version

  // Alpha features are disabled by default and can be enabled by GKE for a particular GKE control plane version.
  // Creating an alpha cluster enables all alpha features by default.
  // Ref: https://cloud.google.com/kubernetes-engine/docs/concepts/feature-gates
  enable_kubernetes_alpha = var.enable_alpha

  service_external_ips_config {
    enabled = true
  }

  node_locations = [
    data.google_compute_zones.available.names[1],
  ]

  node_pool {
    initial_node_count = var.node_count
    management {
      auto_repair  = var.enable_alpha ? false : true
      auto_upgrade = var.enable_alpha ? false : true
    }
    node_config {
      machine_type    = var.instance_type
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

  deletion_protection = false
}

locals {
  kubeconfig = {
    apiVersion = "v1"
    kind       = "Config"
    preferences = {
      colors = true
    }
    current-context = google_container_cluster.primary.name
    contexts = [
      {
        name = google_container_cluster.primary.name
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
          exec = {
            apiVersion         = "client.authentication.k8s.io/v1"
            command            = "gke-gcloud-auth-plugin"
            interactiveMode    = "Never"
            provideClusterInfo = true
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
