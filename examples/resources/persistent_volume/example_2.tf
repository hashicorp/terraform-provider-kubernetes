resource "kubernetes_persistent_volume" "example" {
  metadata {
    name = "example"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      azure_disk {
        caching_mode  = "None"
        data_disk_uri = azurerm_managed_disk.example.id
        disk_name     = "example"
        kind          = "Managed"
      }
    }
  }
}

provider "azurerm" {
  version = ">=2.20.0"
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "example"
  location = "westus2"
}


resource "azurerm_managed_disk" "example" {
  name                 = "example"
  location             = azurerm_resource_group.example.location
  resource_group_name  = azurerm_resource_group.example.name
  storage_account_type = "Standard_LRS"
  create_option        = "Empty"
  disk_size_gb         = "1"
  tags = {
    environment = azurerm_resource_group.example.name
  }
}
