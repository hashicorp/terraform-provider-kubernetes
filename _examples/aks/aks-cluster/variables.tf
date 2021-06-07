variable "kubernetes_version" {
  default = "1.21.1"
}

variable "node_count" {
  default = "3"
}

variable "cluster_name" {
  type = string
}

variable "location" {
  default = "westus2"
}
