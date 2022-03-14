resource "kubernetes_manifest" "test-ingress" {

  manifest = {
    "apiVersion" = "networking.k8s.io/v1beta1"
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
                  "serviceName" = "web"
                  "servicePort" = 80
                }
                "path" = "/"
              },
            ]
          }
        },
      ]
    }
  }
}
