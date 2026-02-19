# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

variable "base_domain" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "kubernetes_version" {
  type    = string
  default = "1.20.2"
}

variable "controller_count" {
  default = 1
}

variable "worker_count" {
  default = 4
}

variable "controller_type" {
  default = "m5a.xlarge"
}

variable "worker_type" {
  default = "m5a.large"
}
