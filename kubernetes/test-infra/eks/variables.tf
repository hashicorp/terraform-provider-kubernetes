# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "cluster_version" {
  default = "1.27"
}

variable "nodes_per_az" {
  default = 1
  type    = number
}

variable "instance_type" {
  default = "m7g.xlarge"
}

variable "az_span" {
  type    = number
  default = 2
  validation {
    condition     = var.az_span > 1
    error_message = "Cluster must span at least 2 AZs"
  }
}

variable "cluster_name" {
  default = ""
}
