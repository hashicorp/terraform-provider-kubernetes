terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "2.62.0"
    }
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "default" {
  name     = var.cluster_name
  location = var.location
}

resource "azurerm_kubernetes_cluster" "default" {
  name                = var.cluster_name
  kubernetes_version  = var.kubernetes_version
  location            = azurerm_resource_group.default.location
  resource_group_name = azurerm_resource_group.default.name
  dns_prefix          = var.cluster_name

  default_node_pool {
    name       = "test"
    node_count = var.node_count
    vm_size    = "Standard_DS2_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}
