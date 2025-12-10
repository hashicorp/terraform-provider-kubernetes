# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

variable "cluster_name" {
  type = string
}
resource "kubernetes_manifest" "test-cfm" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name"      = "test-cf"
      "namespace" = "default"
      "labels" = {
        "parent_cluster" = var.cluster_name
      }
    }
    "data" = {
      "parent_cluster" = var.cluster_name
    }
  }
}
