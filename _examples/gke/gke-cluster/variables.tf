variable "kubernetes_version" {
  default = "1.18"
}

variable "workers_count" {
  default = "3"
}

variable "cluster_name" {
  type = string
}

# This is used to set local variable google_zone.
# This can be replaced with a statically-configured zone, if preferred.
data "google_compute_zones" "available" {
}

locals {
  google_zone = data.google_compute_zones.available.names[0]
}
