# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "kubeconfig" {
  default = "~/.kube/config"
}

variable "name" {
  type        = string
  default     = "staging"
  description = "name of vault instance"
}

variable "namespace" {
  type        = string
  default     = "default"
  description = "namespace to deploy Vault, must already exist"
}

variable "vault_image" {
  type        = string
  default     = "vault:1.4.0"
  description = "container image for vault"
}

variable "vault_k8s_image" {
  type        = string
  default     = "hashicorp/vault-k8s:0.3.0"
  description = "container image for vault-k8s"
}

variable "server_service" {
  type = object({
    port        = number
    targetPort  = number
    annotations = map(string)
  })
  default = {
    port        = 8200
    targetPort  = 8200
    annotations = {}
  }
  description = "headless service parameters"
}
