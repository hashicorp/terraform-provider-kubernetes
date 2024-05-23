# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "customresourcedefinition_cephrbdmirrors_ceph_rook_io" {
  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural}.${var.group}"
    }
    spec = {
      group = var.group
      names = {
        kind = var.kind
        plural = var.plural
      }
      scope = "Namespaced"
      versions = [
        {
          name = var.cr_version
          schema = {
            openAPIV3Schema = {
              properties = {
                spec = {
                  properties = {
                    annotations = {
                      nullable = true
                      type = "object"
                      "x-kubernetes-preserve-unknown-fields" = true
                    }
                    count = {
                      maximum = 100
                      minimum = 1
                      type = "integer"
                    }
                    peers = {
                      properties = {
                        secretNames = {
                          items = {
                            type = "string"
                          }
                          type = "array"
                        }
                      }
                      type = "object"
                    }
                    placement = {
                      nullable = true
                      type = "object"
                      "x-kubernetes-preserve-unknown-fields" = true
                    }
                    priorityClassName = {
                      type = "string"
                    }
                    resources = {
                      nullable = true
                      type = "object"
                      "x-kubernetes-preserve-unknown-fields" = true
                    }
                  }
                  type = "object"
                }
                status = {
                  type = "object"
                  "x-kubernetes-preserve-unknown-fields" = true
                }
              }
              type = "object"
            }
          }
          served = true
          storage = true
          subresources = {
            status = {}
          }
        },
      ]
    }
  }
}
