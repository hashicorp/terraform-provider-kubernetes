variable "kubernetes_version" {
  default = "1.18"
}

variable "workers_count" {
  default = "3"
}

variable "cluster_name" {
  type = string
}

locals {
  google_zone = data.google_compute_zones.available.names[0]
}
