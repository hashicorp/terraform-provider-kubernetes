# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "crd" {
  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind       = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural}.${var.group}"
    }
    spec = {
      group = var.group
      names = {
        kind   = var.kind
        plural = var.plural
      }
      scope = "Namespaced"
      versions = [
        {
          name = var.cr_version
          schema = {
            openAPIV3Schema = {
              type = "object"
              properties = {
                spec = {
                  type = "object"
                  properties = {
                    defaultValues = {
                      type = "object"
                      additionalProperties = {
                        "x-kubernetes-preserve-unknown-fields" = true
                      }
                    }
                  }
                }
              }
            }
          }
          served  = true
          storage = true
        },
      ]
    }
  }
}
