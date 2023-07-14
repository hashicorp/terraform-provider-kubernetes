# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "random_id" "cluster_name" {
  byte_length = 2
  prefix      = "k8s-acc-"
}

locals {
  cluster_name = random_id.cluster_name.hex
}
