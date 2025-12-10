# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

variable "namespace" {
  description = "The namespace where you want to deploy terraform-k8s"
  type        = string
}

variable "tfc_credentials" {
  description = "The file location of your TFC credentials for the terraform-k8s operator"
}

variable "sync_workspace_image" {
  description = "The terraform-k8s operator controller container image"
  default     = "hashicorp/terraform-k8s:0.1.2-alpha"
}

variable "terraformrc_secret_name" {
  description = "Name of Kubernetes secret containing the Terraform CLI Configuration"
  default     = "terraformrc"
}

variable "terraformrc_secret_key" {
  description = "Key of Kubernetes secret containing the Terraform CLI Configuration"
  default     = "credentials"
}

variable "workspace_secrets" {
  description = "Name of Kubernetes secret containing keys and values of sensitive variables "
  default     = "workspacesecrets"
}

variable "access_key_id" {
  description = "Cloud credential held by the workspace-secret"
}

variable "secret_acess_key" {
  description = "Cloud credential held by the workspace-secret"
}