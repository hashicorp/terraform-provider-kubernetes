terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "9.9.9"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "2.42"
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

module "aks-cluster" {
  source                  = "./aks-cluster"
  cluster_name            = local.cluster_name
  location                = var.location
}

module "kubernetes-config" {
  source                  = "./kubernetes-config"
  cluster_id              = module.aks-cluster.cluster_id # creates dependency on cluster creation
  cluster_name            = local.cluster_name
  data_disk_uri           = module.aks-cluster.data_disk_uri
}
