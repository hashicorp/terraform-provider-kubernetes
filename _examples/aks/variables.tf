# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

variable "location" {
  type    = string
  default = "westus2"
}

resource "random_id" "cluster_name" {
  byte_length = 5
}

locals {
  cluster_name = "tf-k8s-${random_id.cluster_name.hex}"
}
