# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test-crd" {
  manifest = {
    "apiVersion" = "apiextensions.k8s.io/v1"
    "kind"       = "CustomResourceDefinition"
    "metadata" = {
      "name" = "${var.plural}.${var.group}"
    }
    "spec" = {
      "group" = var.group
      "names" = {
        "kind"   = var.kind
        "plural" = var.plural
      }
      "scope" = "Namespaced"
      "versions" = [
        {
          "name" = var.cr_version
          "schema" = {
            "openAPIV3Schema" = {
              "properties" = {
                "apiVersion" = {
                  "type" = "string"
                }
                "kind" = {
                  "type" = "string"
                }
                "spec" = {
                  "properties" = {
                    "selfSigned" = {
                      "properties" = {
                        "fooBar" = {
                          "items" = {
                            "type" = "string"
                          }
                          "type" = "array"
                        }
                      }
                      "type" = "object"
                    }
                  }
                  "type" = "object"
                }
              }
              "type" = "object"
            }
          }
          "served"  = true
          "storage" = true
          "subresources" = {
            "status" = {}
          }
        },
      ]
    }
  }
}
