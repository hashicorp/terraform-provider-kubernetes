variable "k8s_node_role_arn" {
  type = list(string)
}

variable "kubeconfig" {
  type = string
}

variable "cluster_ca" {
  type = string
}

variable "cluster_endpoint" {
  type = string
}

