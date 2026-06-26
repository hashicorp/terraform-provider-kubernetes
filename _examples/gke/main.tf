# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.0.3"
    }
    google = {
      source  = "hashicorp/google"
      version = "3.52"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.1.0"
    }
  }
}

# Provider is configured using environment variables: GOOGLE_REGION, GOOGLE_PROJECT, GOOGLE_CREDENTIALS.
# This can be set statically, if preferred. See docs for details.
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#full-reference
provider "google" {}

# Configure kubernetes provider with Oauth2 access token.
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_config
# This fetches a new token, which will expire in 1 hour.
data "google_client_config" "default" {
  depends_on = [module.gke-cluster]
}

# Defer reading the cluster data until the GKE cluster exists.
data "google_container_cluster" "default" {
  name       = local.cluster_name
  depends_on = [module.gke-cluster]
}

provider "kubernetes" {
  host  = "https://${data.google_container_cluster.default.endpoint}"
  token = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(
    data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
  )
}

provider "helm" {
  kubernetes {
    host  = "https://${data.google_container_cluster.default.endpoint}"
    token = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(
      data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
    )
  }
}

module "gke-cluster" {
  source       = "./gke-cluster"
  cluster_name = local.cluster_name
}

module "kubernetes-config" {
  depends_on   = [module.gke-cluster]
  source       = "./kubernetes-config"
  cluster_name = local.cluster_name
}

