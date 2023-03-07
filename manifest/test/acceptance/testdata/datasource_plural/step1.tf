# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_config_a" {
  count = 2
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = "${var.name_prefix}-${count.index}"
      "namespace" = var.namespace
      "labels" = {
        "test" = "terraform"
      }
    }
    "data" = {
      "TEST" = "hello world"
    }
  }
}

resource "kubernetes_manifest" "test_config_b" {
  count = 2
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = "${var.name_prefix}-unlabelled-${count.index}"
      "namespace" = var.namespace
    }
    "data" = {
      "TEST" = "hello world"
    }
  }
}

resource "kubernetes_manifest" "test_config_c" {
  count = 2
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name" = "${var.name_prefix}-annotations-${count.index}"
      "namespace" = var.namespace
      "labels" = {
        "test" = "terraform"
      }
      "annotations" = {
        "test" = "test"
      }
    }
    "data" = {
      "TEST" = "hello world"
    }
  }
}