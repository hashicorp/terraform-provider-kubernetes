variable "location" {
  type = string
  default = "westus2"
}

locals {
  cluster_name = "tf-k8s-${random_id.cluster_name.hex}"
}
