# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

variable "namespace" {
  default = "default"
}

variable "webhook_image" {
  default = "tf-k8s-acc/webhook:latest"
}
