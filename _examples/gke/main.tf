terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
    }
    google = {
      source  = "hashicorp/google"
      version = "3.52"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.0.1"
    }
  }
}

resource "random_id" "cluster_name" {
  byte_length = 5
}

locals {
  cluster_name = "tf-k8s-${random_id.cluster_name.hex}"
}

module "gke-cluster" {
  source                  = "./gke-cluster"
  cluster_name            = local.cluster_name
}

module "kubernetes-config" {
  source                  = "./kubernetes-config"
  cluster_name            = module.gke-cluster.cluster_name
  cluster_id              = module.gke-cluster.cluster_id # creates dependency on cluster creation
  cluster_endpoint        = module.gke-cluster.cluster_endpoint
  cluster_ca_cert         = module.gke-cluster.cluster_ca_cert
}

