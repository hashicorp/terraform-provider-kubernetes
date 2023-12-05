// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

variable "location" {
  type    = string
  default = "West Europe"
}

variable "node_count" {
  type    = number
  default = 2
}

variable "vm_size" {
  type    = string
  default = "Standard_A4_v2"
}

variable "cluster_version" {
  type    = string
  default = "1.27"
}