# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

variable "kubernetes_version" {
  default = "1.18"
}

variable "workers_count" {
  default = "3"
}

variable "cluster_name" {
  type = string
}

variable "location" {
  type = string
}
