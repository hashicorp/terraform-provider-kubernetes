variable "k8s_node_role_arn" {
  type = list(string)
}

variable "cluster_ca_cert" {
  type = string
}

variable "cluster_endpoint" {
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
