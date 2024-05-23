# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "kubernetes_version" {
  default = "1.27"
}

variable "workers_count" {
  default = "3"
}

variable "cluster_name" {
  type = string
}

variable "idp_enabled" {
  type    = bool
  default = false
}
