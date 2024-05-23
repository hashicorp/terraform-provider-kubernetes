# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test-configmap" {

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name"      = "test-config"
      "namespace" = "default"
      "labels" = {
        "app"         = "test-app"
        "environment" = "production"
      }
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
