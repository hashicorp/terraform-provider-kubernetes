
resource "kubernetes_manifest" "test" {

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
          name    = var.cr_version
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
          name    = "${var.cr_version}beta1"
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
