// Copyright (c) HashiCorp, Inc.
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

resource "azurerm_kubernetes_cluster_node_pool" "example" {
  name                  = "spot"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.test.id
  node_count            = var.node_count
  vm_size               = var.vm_size
  priority              = "Spot"
  eviction_policy       = "Delete"
  spot_max_price        = -1
}

resource "local_file" "kubeconfig" {
  content  = azurerm_kubernetes_cluster.test.kube_config_raw
  filename = "${path.module}/kubeconfig"
}
