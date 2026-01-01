# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

data "kubernetes_resources" "example"{
    kind = "ConfigMap"
    api_version = "v1"
    namespace = var.namespace
    label_selector = var.label_selector
    limit = var.limit
}