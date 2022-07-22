locals {
  random_prefix = "${var.prefix}-${random_id.tf-k8s-acc.hex}"
}

provider "azurerm" {
  features {}
}

data "azurerm_kubernetes_service_versions" "current" {
  location       = var.location
  version_prefix = var.kubernetes_version
}

resource "random_id" "tf-k8s-acc" {
  byte_length = 3
}

resource "azurerm_resource_group" "tf-k8s-acc" {
  name     = "${local.random_prefix}-rsg"
  location = var.location
}

resource "azurerm_route_table" "tf-k8s-acc" {
  name                = "${local.random_prefix}-rt"
  location            = azurerm_resource_group.tf-k8s-acc.location
  resource_group_name = azurerm_resource_group.tf-k8s-acc.name

  route {
    name                   = "default"
    address_prefix         = "10.100.0.0/14"
    next_hop_type          = "VirtualAppliance"
    next_hop_in_ip_address = "10.10.1.1"
  }
}

resource "azurerm_virtual_network" "tf-k8s-acc" {
  name                = "${local.random_prefix}-network"
  location            = azurerm_resource_group.tf-k8s-acc.location
  resource_group_name = azurerm_resource_group.tf-k8s-acc.name
  address_space       = ["10.1.0.0/16"]
}

resource "azurerm_subnet" "tf-k8s-acc" {
  name                 = "${local.random_prefix}-internal"
  resource_group_name  = azurerm_resource_group.tf-k8s-acc.name
  address_prefixes     = ["10.1.0.0/24"]
  virtual_network_name = azurerm_virtual_network.tf-k8s-acc.name
}

resource "azurerm_subnet_route_table_association" "tf-k8s-acc" {
  subnet_id      = azurerm_subnet.tf-k8s-acc.id
  route_table_id = azurerm_route_table.tf-k8s-acc.id
}

resource "azurerm_kubernetes_cluster" "tf-k8s-acc" {
  name                = "${local.random_prefix}-cluster"
  resource_group_name = azurerm_resource_group.tf-k8s-acc.name
  location            = azurerm_resource_group.tf-k8s-acc.location
  dns_prefix          = "${local.random_prefix}-cluster"
  kubernetes_version  = data.azurerm_kubernetes_service_versions.current.latest_version

  # Uncomment to enable SSH access to nodes
  #
  # linux_profile {
  #   admin_username = "acctestuser1"
  #   ssh_key {
  #     key_data = "${file(var.public_ssh_key_path)}"
  #   }
  # }

  default_node_pool {
    name            = "agentpool"
    node_count      = var.workers_count
    vm_size         = var.workers_type
    os_disk_size_gb = 30

    # Required for advanced networking
    vnet_subnet_id = azurerm_subnet.tf-k8s-acc.id
  }


  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin = "azure"
  }
}

resource "local_file" "kubeconfig" {
  content  = azurerm_kubernetes_cluster.tf-k8s-acc.kube_config_raw
  filename = "${path.module}/kubeconfig"
}

