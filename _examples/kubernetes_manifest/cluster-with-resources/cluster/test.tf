# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "location" {
  type = string
}

data "azurerm_kubernetes_service_versions" "current" {
  location = var.location
}
resource "azurerm_resource_group" "test" {
  name     = "k8s-alpha-test"
  location = var.location
}
resource "azurerm_kubernetes_cluster" "test" {
  name                = "k8s-alpha-aks"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  dns_prefix          = "k8s-alpha-aks"

  kubernetes_version = data.azurerm_kubernetes_service_versions.current.latest_version

  default_node_pool {
    name            = "default"
    vm_size         = "Standard_D2_v2"
    os_disk_size_gb = 30
    node_count      = 2
    type            = "AvailabilitySet"
  }

  identity {
    type = "SystemAssigned"
  }

  role_based_access_control {
    enabled = true
  }

  tags = {
    environment = "k8s-alpha"
  }
}
resource "local_file" "kubeconfig" {
  content  = azurerm_kubernetes_cluster.test.kube_config_raw
  filename = "kubeconfig.test"
}

output "host" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.host
}

output "cluster_ca_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.test.kube_config.0.cluster_ca_certificate)
}

output "client_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.test.kube_config.0.client_certificate)
}

output "client_key" {
  value = base64decode(azurerm_kubernetes_cluster.test.kube_config.0.client_key)
}

output "cluster_resource_group" {
  value = azurerm_resource_group.test.name
}

output "cluster_name" {
  value = azurerm_kubernetes_cluster.test.name
}
