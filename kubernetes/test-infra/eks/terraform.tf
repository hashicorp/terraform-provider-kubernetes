# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0


terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0.0, < 6.0.0"
    }
  }
}
