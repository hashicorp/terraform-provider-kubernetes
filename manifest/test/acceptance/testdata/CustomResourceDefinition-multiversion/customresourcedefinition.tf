# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test1" {

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind       = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural1}.${var.group1}"
    }
    spec = {
      group = var.group1
      names = {
        kind   = var.kind1
        plural = var.plural1
      }
      scope = "Namespaced"
      versions = [
        {
          name    = var.cr_version1
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
        },
        {
          name    = "${var.cr_version1}beta1"
          served  = true
          storage = false
          schema = {
            openAPIV3Schema = {
              type = "object"
              properties = {
                data = {
                  type = "string"
                }
                otherData = {
                  type = "string"
                }
                refs = {
                  type = "number"
                }
              }
            }
          }
        }
      ]
    }
  }
}

resource "kubernetes_manifest" "test2" {

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind       = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural2}.${var.group2}"
    }
    spec = {
      group = var.group2
      names = {
        kind   = var.kind2
        plural = var.plural2
      }
      scope = "Namespaced"
      versions = [
        {
          name    = "${var.cr_version2}alpha1"
          served  = true
          storage = true
          schema = {
            openAPIV3Schema = {
              type = "object"
              properties = {
                refs = {
                  type = "number"
                }
              }
            }
          }
        }
      ]
    }
  }
}
