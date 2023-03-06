provider "azurerm" {
  features {}
}

variable "location" {
  type = string
  default = "West Europe"
}

variable "node_count" {
  type = number
  default = 2
}

variable "vm_size" {
  type = string
  default = "Standard_A4_v2"
}

variable "kubernetes_version" {
  type = string
  default = "1.25.5"
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
  kubernetes_version  = var.kubernetes_version

  default_node_pool {
    name       = "default"
    node_count = var.node_count
    vm_size    = var.vm_size
    //vm_size    = "Standard_A4_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "local_file" "kubeconfig" {
  content  = azurerm_kubernetes_cluster.test.kube_config_raw
  filename = "${path.module}/kubeconfig"
}

output "kubeconfig" {
  value = azurerm_kubernetes_cluster.test.kube_config_raw

  sensitive = true
}

output "cluster_name" {
  value = "test-aks-${random_pet.name.id}"
}

