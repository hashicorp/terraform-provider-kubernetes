terraform {
￼  required_providers {
￼    kubernetes = {
￼      source  = "hashicorp/kubernetes"
#￼      version = "2.0"
￼    }
￼    google = {
￼      source  = "hashicorp/azure"
￼      version = "2.42"
￼    }
￼    helm = {
￼      source  = "hashicorp/helm"
￼      version = "2.0.1"
￼    }
￼  }
}

resource "random_id" "cluster_name" {
  byte_length = 5
}


module "aks-cluster" {
  source                  = "./aks-cluster"
  cluster_name            = local.cluster_name
}

module "kubernetes-config" {
  source                  = "./kubernetes-config"
  cluster_name            = module.aks-cluster.cluster_name
  cluster_id              = module.aks-cluster.cluster_id # creates dependency on cluster creation
  cluster_endpoint        = module.aks-cluster.cluster_endpoint
  cluster_ca_cert         = module.aks-cluster.cluster_ca_cert
}

