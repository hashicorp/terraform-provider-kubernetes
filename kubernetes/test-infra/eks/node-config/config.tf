
provider "kubernetes" {
  config_path = "${var.kubeconfig_path}"
}

resource "kubernetes_config_map" "name" {
  metadata {
    name      = "aws-auth"
    namespace = "kube-system"
  }
  data {
    mapRoles = <<MAPROLES
    - rolearn: ${var.k8s_node_role_arn}
      username: system:node:{{EC2PrivateDNSName}}
      groups:
        - system:bootstrappers
        - system:nodes
MAPROLES
}
}
