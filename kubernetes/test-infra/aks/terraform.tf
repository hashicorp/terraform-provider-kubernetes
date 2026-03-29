# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0


terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.0.0, < 4.0.0"
    }
  }
}
