# https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_config
data "google_client_config" "default" {
}

data "google_container_cluster" "default" {
  name = var.cluster_name
}

#provider "kubernetes" {
#  host                   = "https://${data.google_container_cluster.default.endpoint}"
#  cluster_ca_certificate = base64decode(
#    data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
#  )
#  exec {
#    command     = "gcloud"
#    api_version = "client.authentication.k8s.io/v1alpha1"
#    args        = ["container", "clusters", "get-credentials", var.cluster_name,
#                   "--zone", data.google_client_config.default.zone, "--project",
#                   data.google_client_config.default.project
#                  ]
#  }
#}
#
#provider "helm" {
#  kubernetes {
#    host                   = "https://${data.google_container_cluster.default.endpoint}"
#    cluster_ca_certificate = base64decode(
#      data.google_container_cluster.default.master_auth[0].cluster_ca_certificate,
#    )
#    exec {
#      command     = "gcloud"
#      api_version = "client.authentication.k8s.io/v1alpha1"
#      args        = ["container", "clusters", "get-credentials", var.cluster_name, "--zone",
#                     data.google_client_config.default.zone, "project",
#                     data.google_client_config.default.project
#                    ]
#    }
#  }
#}
#
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

#resource "kubernetes_namespace" "test" {
#  metadata {
#    name = "test"
#  }
#}
#
#resource "helm_release" "nginx_ingress" {
#  namespace  = kubernetes_namespace.test.metadata.0.name
#  wait       = true
#  timeout    = 600
#
#  name       = "ingress-nginx"
#
#  repository = "https://kubernetes.github.io/ingress-nginx"
#  chart      = "ingress-nginx"
#  version    = "v3.30.0"
#}
