variable "k8s_node_role_arn" {
  type = string
}

variable "cluster_name" {
  type = string
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
