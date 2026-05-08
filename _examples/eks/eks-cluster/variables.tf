# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

variable "cluster_name" {
  type = string
}

variable "kubernetes_version" {
  type    = string
  default = "1.27"
}
