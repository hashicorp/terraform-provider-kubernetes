#
# Variables Configuration
#
variable "region" {
  default = "us-west-1"
  type    = "string"
}

variable "kubernetes_version" {
  type    = "string"
  default = "1.11"
}

variable "workers_count" {
  default = 2
}

variable "workers_type" {
  type    = "string"
  default = "m4.large"
}
