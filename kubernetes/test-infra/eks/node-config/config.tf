provider "kubernetes" {
  config_path      = "${local_file.kubeconfig.filename}"
  load_config_file = true
}

resource "local_file" "kubeconfig" {
  content  = "${var.kubeconfig}"
  filename = "${path.module}/kubeconfig"
}

locals {
  mapped_role_format = <<MAPPEDROLE
- rolearn: %s
  username: system:node:{{EC2PrivateDNSName}}
  groups:
    - system:bootstrappers
    - system:nodes
MAPPEDROLE
}

resource "kubernetes_config_map" "name" {
  metadata {
    name = "aws-auth"
    namespace = "kube-system"
  }

  data {
    mapRoles = "${join("\n", formatlist(local.mapped_role_format, var.k8s_node_role_arn))}"
  }
}
