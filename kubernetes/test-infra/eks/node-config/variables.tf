variable "k8s_node_role_arn" {
  type = list(string)
}

variable "cluster_ca" {
  type = string
}

variable "cluster_endpoint" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "cluster_oidc_issuer_url" {
  type = string
}
