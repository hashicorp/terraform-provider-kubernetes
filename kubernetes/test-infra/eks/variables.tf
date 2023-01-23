variable "cluster_version" {
  default = "1.23"
}

variable "nodes_per_az" {
  default = 1
  type    = number
}

variable "instance_type" {
  default = "m5.large"
}

variable "az_span" {
  type    = number
  default = 3
  validation {
    condition     = var.az_span > 1
    error_message = "Cluster must span at least 2 AZs"
  }
}
