# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.1.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.39.0"
    }
  }
}
