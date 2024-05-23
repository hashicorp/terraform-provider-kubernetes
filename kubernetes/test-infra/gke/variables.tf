# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "cluster_version" {
  default = ""
}

variable "node_count" {
  default = "1"
}

variable "instance_type" {
  default = "e2-standard-2"
}

variable "enable_alpha" {
  default = false
}

variable "cluster_name" {
  default = ""
}
