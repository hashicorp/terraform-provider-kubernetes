resource "kubernetes_manifest" "crd" {
  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind       = "CustomResourceDefinition"
    metadata = {
      name = "${var.plural}.${var.group}"
    }
    spec = {
      group = "${var.group}"
      names = {
        kind   = "${var.kind}"
        plural = "${var.plural}"
      }
      scope = "Namespaced"
      versions = [
        {
          name    = "${var.cr_version}"
          served  = true
          storage = true
          schema = {
            openAPIV3Schema = {
              type = "object"
              properties = {
                spec = {
                  type = "object"
                  properties = {
                    patchStrategicMerge = {
                      type = "object"
                      properties = {
                        containers = {
                          type = "array"
                          "x-kubernetes-list-type" = "map"
                          "x-kubernetes-list-map-keys" = ["name"]
                          items = {
                            type = "object"
                            required = ["name"]
                            properties = {
                              name = {
                                type = "string"
                              }
                              image = {
                                type = "string"
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        },
      ]
    }
  }
}
