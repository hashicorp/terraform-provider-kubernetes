# Configure kubernetes provider with Oauth2 access token.
# This fetches a new token, which will expire in 1 hour.
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_config
data "google_client_config" "default" {
}

data "google_container_cluster" "default" {
  name = var.cluster_name
}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.default.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(
    data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
  )
}

provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.default.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(
      data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
    )
  }
}

resource "local_file" "kubeconfig" {
  filename          = "./kubeconfig"
  sensitive_content = templatefile("${path.module}/kubeconfig.tpl", {
    ca_cert         = data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
    cluster_name    = var.cluster_name,
    endpoint        = "https://${data.google_container_cluster.default.endpoint}"
    project         = data.google_client_config.default.project
    zone            = data.google_client_config.default.zone
  })
}

resource "kubernetes_namespace" "test" {
  metadata {
    name = "test"
  }
}

resource "helm_release" "nginx_ingress" {
  namespace  = kubernetes_namespace.test.metadata.0.name
  wait       = true
  timeout    = 600

  name       = "ingress-nginx"

  repository = "https://kubernetes.github.io/ingress-nginx"
  chart      = "ingress-nginx"
  version    = "v3.30.0"
}
