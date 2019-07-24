variable "prefix" {
  description = "A prefix used for all resources in this example"
  default     = "tf-k8s-acc"
}

variable "location" {
  default     = "West Europe"
  description = "The Azure Region in which all resources in this example should be provisioned"
}

variable "kubernetes_version" {
  type = string
}

variable "workers_count" {
  type    = string
  default = 2
}

variable "workers_type" {
  type    = string
  default = "Standard_DS4_v2"
}

variable "aks_client_id" {
  description = "The Client ID for the Service Principal to use for this Managed Kubernetes Cluster"
}

variable "aks_client_secret" {
  description = "The Client Secret for the Service Principal to use for this Managed Kubernetes Cluster"
}

# Uncomment to enable SSH access to nodes
#
# variable "public_ssh_key_path" {
#   description = "The Path at which your Public SSH Key is located. Defaults to ~/.ssh/id_rsa.pub"
#   default     = "~/.ssh/id_rsa.pub"
#}
