# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Regression test: empty object with x-kubernetes-preserve-unknown-fields
# This tests the DynamicPseudoType preservation fix.
# Without the fix, this would fail with:
#   "Provider produced inconsistent result after apply"
#   "wrong final value type: incorrect object attributes"

resource "kubernetes_manifest" "test" {
  manifest = {
    apiVersion = var.group_version
    kind       = var.kind
    metadata = {
      name      = var.name
      namespace = var.namespace
    }
    spec = {
      count     = 100
      resources = {} # Empty object - triggers DynamicPseudoType bug without fix
    }
  }
}
