#
# Variables Configuration
#
variable "kubernetes_version" {
  type    = string
  default = "1.18"
}

variable "workers_count" {
  default = 2
}

variable "workers_type" {
  type    = string
  default = "m4.large"
}
