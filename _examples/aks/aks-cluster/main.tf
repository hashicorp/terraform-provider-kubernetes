resource "azurerm_resource_group" "test" {
  name     = var.cluster_name
  location = var.location
}

resource "azurerm_kubernetes_cluster" "test" {
  name                = var.cluster_name
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  dns_prefix          = var.cluster_name

  default_node_pool {
    name       = "default"
    node_count = 1
    vm_size    = "Standard_DS2_v2"
    #vm_size    = "Standard_A2_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "local_file" "kubeconfig" {
  content = azurerm_kubernetes_cluster.test.kube_config_raw
  filename = "${path.root}/kubeconfig"
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
