terraform {
  required_providers {
    kubernetes-local = {
      source = "localhost/test/kubernetes"
      version = "9.9.9"
    }
    helm = {
      source  = "localhost/test/helm"
      version = "9.9.9"
    }
  }
}

resource "kubernetes_namespace" "test" {
  provider = kubernetes-local
  metadata {
    name = "test"
  }
}

resource helm_release nginx_ingress {
  wait       = false
  name       = "ingress-nginx"

  repository = "https://kubernetes.github.io/ingress-nginx"
  chart      = "ingress-nginx"
  version    = "v3.24.0"

  set {
    name  = "controller.updateStrategy.rollingUpdate.maxUnavailable"
    value = "1"
  }
  set {
    name  = "controller.replicaCount"
    value = "2"
  }
  set_sensitive {
    name = "controller.maxmindLicenseKey"
    value = "testSensitiveValue"
  }
}
