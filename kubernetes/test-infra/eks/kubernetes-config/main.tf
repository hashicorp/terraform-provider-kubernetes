resource "kubernetes_config_map" "name" {
depends_on = [var.cluster_name]
  metadata {
    name      = "aws-auth"
    namespace = "kube-system"
  }

  data = {
    mapRoles = join(
      "\n",
      formatlist(local.mapped_role_format, var.k8s_node_role_arn),
    )
  }
}

resource "kubernetes_namespace" "test" {
depends_on = [var.cluster_name]
  metadata {
    name = "test"
  }
}

resource helm_release nginx_ingress {
depends_on = [var.cluster_name]
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}
