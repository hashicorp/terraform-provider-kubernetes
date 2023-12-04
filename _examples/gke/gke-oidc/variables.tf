# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "cluster_name" {
  description = "Name of target GKE cluster"
  type        = string
}

variable "gke_location" {
  description = "Location of target GKE cluster"
  type        = string
}

variable "oidc_audience" {
  description = "Audience value as configured in TFC / TFE environment variable"
  type        = string
  default     = "kubernetes"
}

variable "odic_issuer_uri" {
  description = "Base URL of TFC / TFE endpoint (default to public TFC)"
  type        = string
  default     = "https://app.terraform.io"
}

variable "oidc_user_claim" {
  description = "Token claim to extract user name from (defaults to 'sub')"
  type        = string
  default     = "sub"
}

variable "oidc_group_claim" {
  description = "Token claim to extract the group membership from (defaults to 'terraform_organization_name')"
  type        = string
  default     = "terraform_organization_name"
}

variable "TFE_CA_cert" {
  description = "CA Certificate for the HTTPS API endpoint of TFE"
  type        = string
  default     = null
}

variable "rbac_oidc_group_name" {
  description = "Name of OIDC group (according to 'oidc_group_claim') to be granted the role designated by 'var.rbac_group_cluster_role'"
  type = string
}

variable "rbac_group_cluster_role" {
  description = "Kubernetes role to be bound to the OIDC group designated by 'var.rbac_oidc_group_name'"
  default = "cluster-admin"
}
