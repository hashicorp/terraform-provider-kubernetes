# Configure kubernetes provider with Oauth2 access token.
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_config
# This fetches a new token, which will expire in 1 hour.
data "google_client_config" "default" {
}

provider "kubernetes" {
  host = var.cluster_endpoint
  token = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(var.cluster_ca_cert)
}

resource "kubernetes_namespace" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_deployment" "test" {
  metadata {
    name = "test"
    namespace= kubernetes_namespace.test.metadata.0.name
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        TestLabelOne   = "one"
      }
    }
    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
        }
      }
      spec {
        container {
          image = "nginx:1.19.4"
          name  = "tf-acc-test"

          resources {
            limits = {
              memory = "512M"
              cpu = "1"
            }
            requests = {
              memory = "256M"
              cpu = "50m"
            }
          }
        }
      }
    }
  }
}

provider "helm" {
  kubernetes {
    host = var.cluster_endpoint
    token = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(var.cluster_ca_cert)
  }
}

resource helm_release nginx_ingress {
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}

data "template_file" "kubeconfig" {
  template = file("${path.module}/kubeconfig-template.yaml")

  vars = {
    cluster_name    = var.cluster_name
    endpoint        = var.cluster_endpoint
    cluster_ca      = var.cluster_ca_cert
    cluster_token   = data.google_client_config.default.access_token
  }
}

resource "local_file" "kubeconfig" {
  depends_on = [var.cluster_id]
  content  = data.template_file.kubeconfig.rendered
  filename = "${path.root}/kubeconfig"
}

