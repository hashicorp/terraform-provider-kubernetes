# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test-ingress" {

  manifest = {
    "apiVersion" = "networking.k8s.io/v1"
    "kind"       = "Ingress"
    "metadata" = {
      "annotations" = {
        "nginx.ingress.kubernetes.io/rewrite-target" = "/$1"
      }
      "name"      = "example-ingress"
      "namespace" = "default"
    }
    "spec" = {
      "rules" = [
        {
          "host" = "hello-world.info"
          "http" = {
            "paths" = [
              {
                "backend" = {
                  "service" = {
                    "name" = "test"
                    "port" = {
                      "number" = "80"
                    }
                  }
                }
                "path"     = "/"
                "pathType" = "Prefix"
              },
            ]
          }
        },
      ]
    }
  }
}
