provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = local.cluster_name
  location = var.location
}

resource "azurerm_kubernetes_cluster" "test" {
  name                = local.cluster_name
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  dns_prefix          = local.cluster_name

  default_node_pool {
    name       = "default"
    node_count = 1
    vm_size    = "Standard_DS2_v2"
  }

  identity {
    type = "SystemAssigned"
  }

  addon_profile {
    aci_connector_linux {
      enabled = false
    }

    azure_policy {
      enabled = false
    }

    http_application_routing {
      enabled = false
    }

    kube_dashboard {
      enabled = true
    }

    oms_agent {
      enabled = false
    }
  }
}

resource "local_file" "kubeconfig" {
  content = azurerm_kubernetes_cluster.test.kube_config_raw
  filename = "${path.module}/kubeconfig"
}

resource "azurerm_managed_disk" "test" {
  name                 = "testdisk"
  location             = azurerm_resource_group.test.location
  resource_group_name  = azurerm_resource_group.test.name
  storage_account_type = "Standard_LRS"
  create_option        = "Empty"
  disk_size_gb         = "1"
  tags = {
    environment = azurerm_resource_group.test.name
  }
}
