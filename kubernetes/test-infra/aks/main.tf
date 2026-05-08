// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

provider "azurerm" {
  features {}
}

resource "random_pet" "name" {}

resource "azurerm_resource_group" "test_group" {
  name     = "test-aks-${random_pet.name.id}"
  location = var.location
}

resource "azurerm_kubernetes_cluster" "test" {
  name                = "test-aks-${random_pet.name.id}"
  location            = azurerm_resource_group.test_group.location
  resource_group_name = azurerm_resource_group.test_group.name
  dns_prefix          = "test"
  kubernetes_version  = var.cluster_version

  default_node_pool {
    name       = "default"
    node_count = var.node_count
    vm_size    = var.vm_size
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "local_file" "kubeconfig" {
  content  = azurerm_kubernetes_cluster.test.kube_config_raw
  filename = "${path.module}/kubeconfig"
}
