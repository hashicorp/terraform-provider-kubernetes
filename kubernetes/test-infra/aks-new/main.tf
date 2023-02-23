provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test_group" {
  name     = "test-group"
  location = "West Europe"
}

resource "azurerm_kubernetes_cluster" "test" {
  name                = "test-aks1"
  location            = azurerm_resource_group.test_group.location
  resource_group_name = azurerm_resource_group.test_group.name
  dns_prefix          = "test"

  default_node_pool {
    name       = "default"
    node_count = 2
    vm_size    = "Standard_A4_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}

output "kube_config" {
  value = azurerm_kubernetes_cluster.test.kube_config_raw

  sensitive = true
}

