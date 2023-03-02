# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "linode" {
  // Provider settings to be provided via ENV variables
}

locals {
  namePrefix = "tf-acc-test"
}

resource "random_id" "cluster_label" {
  byte_length = 10
}

variable "kubernetes_version" {
  type    = string
  default = "1.17"
}

variable "workers_type" {
  type    = string
  default = "g6-standard-2"
}

variable "workers_count" {
  type    = number
  default = 3
}

resource "linode_lke_cluster" "cluster" {
  label       = "${local.namePrefix}-${random_id.cluster_label.id}"
  region      = "us-east"
  k8s_version = var.kubernetes_version
  tags        = ["acc-test"]

  pool {
    type  = var.workers_type
    count = var.workers_count
  }
}

resource "local_file" "kubeconfig" {
  content  = base64decode(linode_lke_cluster.cluster.kubeconfig)
  filename = "kubeconfig"
}

output "kubeconfig_path" {
  value = "${path.cwd}/${local_file.kubeconfig.filename}"
}
