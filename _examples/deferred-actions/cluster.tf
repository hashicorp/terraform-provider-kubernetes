# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    kind = {
      source = "tehcyx/kind"
    }
    kubernetes = {}
  }
}

resource "kind_cluster" "demo" {
  name = "demo-cluster"
}

provider "kubernetes" {
  host                   = kind_cluster.demo.endpoint
  cluster_ca_certificate = kind_cluster.demo.cluster_ca_certificate
  client_certificate     = kind_cluster.demo.client_certificate
  client_key             = kind_cluster.demo.client_key
}
