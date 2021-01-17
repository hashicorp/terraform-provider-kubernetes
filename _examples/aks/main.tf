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

provider "kubernetes" {
  host                   = module.aks-cluster.endpoint
  client_key             = base64decode(module.aks-cluster.client_key)
  client_certificate     = base64decode(module.aks-cluster.client_cert)
  cluster_ca_certificate = base64decode(module.aks-cluster.ca_cert)
}

provider "helm" {
  kubernetes {
    host                   = module.aks-cluster.endpoint
    client_key             = base64decode(module.aks-cluster.client_key)
    client_certificate     = base64decode(module.aks-cluster.client_cert)
    cluster_ca_certificate = base64decode(module.aks-cluster.ca_cert)
  }
}

provider "azurerm" {
  features {}
}

module "aks-cluster" {
  providers               = { azurerm = azurerm }
  source                  = "./aks-cluster"
  cluster_name            = local.cluster_name
  location                = var.location
}

module "kubernetes-config" {
  providers               = { kubernetes = kubernetes, helm = helm }
  depends_on              = [module.aks-cluster]
  source                  = "./kubernetes-config"
  cluster_name            = local.cluster_name
  data_disk_uri           = module.aks-cluster.data_disk_uri
}
