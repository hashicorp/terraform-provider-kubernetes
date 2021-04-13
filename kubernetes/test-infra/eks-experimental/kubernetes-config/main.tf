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

# For this resource, we need to explicitly establish the dependency on the cluster API, because the dependency is not yet present in this file.
# https://github.com/terraform-aws-modules/terraform-aws-eks/blob/31ad394dbc61390dc46643b571249a2b670e9caa/kubectl.tf
resource "kubernetes_namespace" "test" {
  depends_on  = [var.cluster_name]
  provider = kubernetes-local
  metadata {
    name = "test"
  }
}

resource helm_release nginx_ingress {
  wait       = true
  timeout    = 600

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
