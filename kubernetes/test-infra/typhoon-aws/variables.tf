variable "base_domain" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "controller_count" {
  default = 1
}

variable "worker_count" {
  default = 4
}

variable "controller_type" {
  default = "m5a.xlarge"
}

variable "worker_type" {
  default = "m5a.large"
}
