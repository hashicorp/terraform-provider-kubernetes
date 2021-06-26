provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
}

resource "kubernetes_manifest" "test-pod" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Pod"
    "metadata" = {
      "name"      = "label-demo"
      "namespace" = "default"
      "labels" = {
        "app"         = "nginx"
        "environment" = "production"
      }
    }
    "spec" = {
      "containers" = [
        {
          "image" = "nginx:1.7.9"
          "name"  = "nginx"
          "ports" = [
            {
              "containerPort" = 80
              "protocol"      = "TCP"
            },
          ]
          env = [
            {
              "name" = "VAR1"
              "valueFrom" = {
                "fieldRef" = {
                  "fieldPath" = "metadata.namespace"
                }
              }
            },
            {
              "name"  = "VAR2"
              "value" = "http://127.0.0.1:8200"
            },
          ]
        },
      ]
    }
  }

}
