# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "google" {
  // Provider settings to be provided via ENV variables
}

data "google_compute_zones" "available" {
}

resource "random_id" "cluster_name" {
  byte_length = 10
}

resource "random_id" "username" {
  byte_length = 14
}

resource "random_id" "password" {
  byte_length = 16
}

variable "kubernetes_version" {
  default = ""
}

variable "workers_count" {
  default = "3"
}

data "google_container_engine_versions" "supported" {
  location       = data.google_compute_zones.available.names[0]
  version_prefix = var.kubernetes_version
}

# If the result is empty '[]', the GKE default_cluster_version will be used.
output "available_master_versions_matching_user_input" {
  value = data.google_container_engine_versions.supported.valid_master_versions
}

# Shared network for GKE cluster and Filestore to use.
resource "google_compute_network" "vpc" {
  name                    = "shared"
  auto_create_subnetworks = true
}

resource "google_container_cluster" "primary" {
  name               = "tf-acc-test-${random_id.cluster_name.hex}"
  location           = data.google_compute_zones.available.names[0]
  network            = google_compute_network.vpc.name
  initial_node_count = var.workers_count
  min_master_version = data.google_container_engine_versions.supported.latest_master_version
  # node version must match master version
  # https://www.terraform.io/docs/providers/google/r/container_cluster.html#node_version
  node_version = data.google_container_engine_versions.supported.latest_master_version

  node_locations = [
    data.google_compute_zones.available.names[1],
  ]

  master_auth {
    username = random_id.username.hex
    password = random_id.password.hex
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


resource "google_filestore_instance" "test" {
  name = "test-nfs-server"
  tier = "STANDARD"
  zone = "us-west1-a"

  file_shares {
    capacity_gb = 1024
    name        = "vol1"
  }

  networks {
    network = google_compute_network.vpc.name
    modes   = ["MODE_IPV4"]
  }
}

resource "local_file" "kubeconfig" {
  content  = templatefile("${path.module}/kubeconfig-template.yaml",{
    cluster_name    = google_container_cluster.primary.name
    user_name       = google_container_cluster.primary.master_auth[0].username
    user_password   = google_container_cluster.primary.master_auth[0].password
    endpoint        = google_container_cluster.primary.endpoint
    cluster_ca      = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
    client_cert     = google_container_cluster.primary.master_auth[0].client_certificate
    client_cert_key = google_container_cluster.primary.master_auth[0].client_key
  })
  filename = "${path.module}/kubeconfig"
}

provider "kubernetes" {
  version          = "1.11.2"
  load_config_file = "false"

  host = google_container_cluster.primary.endpoint

  username               = google_container_cluster.primary.master_auth[0].username
  password               = google_container_cluster.primary.master_auth[0].password
  client_certificate     = base64decode(google_container_cluster.primary.master_auth[0].client_certificate)
  client_key             = base64decode(google_container_cluster.primary.master_auth[0].client_key)
  cluster_ca_certificate = base64decode(google_container_cluster.primary.master_auth[0].cluster_ca_certificate)
}

resource "kubernetes_namespace" "example" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_storage_class" "nfs" {
  metadata {
    name = "filestore"
  }
  reclaim_policy      = "Retain"
  storage_provisioner = "nfs"
}

resource "kubernetes_persistent_volume" "example" {
  metadata {
    name = "nfs-volume"
  }
  spec {
    capacity = {
      storage = "1T"
    }
    storage_class_name = kubernetes_storage_class.nfs.metadata[0].name
    access_modes       = ["ReadWriteMany"]
    persistent_volume_source {
      nfs {
        server = google_filestore_instance.test.networks[0].ip_addresses[0]
        path   = "/${google_filestore_instance.test.file_shares[0].name}"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim" "example" {
  metadata {
    name      = "mariadb-data"
    namespace = "test"
  }
  spec {
    access_modes       = ["ReadWriteMany"]
    storage_class_name = kubernetes_storage_class.nfs.metadata[0].name
    volume_name        = kubernetes_persistent_volume.example.metadata[0].name
    resources {
      requests = {
        storage = "1T"
      }
    }
  }
}

resource "kubernetes_deployment" "mariadb" {
  metadata {
    name      = "mariadb-example"
    namespace = "test"
    labels = {
      mylabel = "MyExampleApp"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        mylabel = "MyExampleApp"
      }
    }

    template {
      metadata {
        labels = {
          mylabel = "MyExampleApp"
        }
      }

      spec {
        container {
          image = "mariadb:10.5.2"
          name  = "example"

          env {
            name  = "MYSQL_RANDOM_ROOT_PASSWORD"
            value = true
          }

          resources {
            limits = {
              cpu    = "0.5"
              memory = "512Mi"
            }
            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }

          volume_mount {
            mount_path = "/var/lib/mysql"
            name       = "mariadb-data"
          }
        }
        volume {
          name = "mariadb-data"
          persistent_volume_claim {
            claim_name = "mariadb-data"
          }
        }
      }
    }
  }
}

output "node_version" {
  value = google_container_cluster.primary.node_version
}

output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}
