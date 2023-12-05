# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "location" {
  type    = string
  default = "West Europe"
}

/*
  This creates a simple cluster on AKS.
*/
provider "azurerm" {
  version = ">=2.20.0"
  features {}
}

module "cluster" {
  source   = "./cluster"
  location = var.location
}

/*
  Here we create the Kubernetes resources on the AKS cluster.
  
  IMPORTANT: there is no explicit or implicit way to express dependency 
  of the Kubernetes resource on the AKS resource being present.
  You must split the apply into two operations. See README.md
*/

module "manifests" {
  source       = "./manifests"
  cluster_name = module.cluster.cluster_name
}
