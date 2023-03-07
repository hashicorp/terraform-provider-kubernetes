# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test-crd" {

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind       = "CustomResourceDefinition"
    metadata = {
      name = "testcrds.hashicorp.com"
      labels = {
        app = "test"
      }
    }
    spec = {
      group = "hashicorp.com"
      names = {
        kind     = "TestCrd"
        plural   = "testcrds"
        singular = "testcrd"
        listKind = "TestCrds"
      }
      scope = "Namespaced"
      conversion = {
        strategy = "None"
      }
      versions = [{
        name    = "v1"
        served  = true
        storage = true
        schema = {
          openAPIV3Schema = {
            type = "object"
            properties = {
              data = {
                type = "string"
              }
              refs = {
                type = "number"
              }
            }
          }
        }
      }]
    }
  }
}
