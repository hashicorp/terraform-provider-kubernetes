---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_persistent_volume"
description: |-
  A Persistent Volume (PV) is a piece of networked storage in the cluster that has been provisioned by an administrator.
---

# <no value>

The resource provides a piece of networked storage in the cluster provisioned by an administrator. It is a resource in the cluster just like a node is a cluster resource. Persistent Volumes have a lifecycle independent of any individual pod that uses the PV. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)

<no value>

## Example Usage

```terraform
resource "kubernetes_persistent_volume" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    capacity = {
      storage = "2Gi"
    }
    access_modes = ["ReadWriteMany"]
    persistent_volume_source {
      vsphere_volume {
        volume_path = "/absolute/path"
      }
    }
  }
}
```

## Example: Persistent Volume using Azure Managed Disk

```terraform
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
```

## Import

Persistent Volume can be imported using its name, e.g.

```
$ terraform import kubernetes_persistent_volume.example terraform-example
```
