# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "cluster_name" {
  type = string
}

variable "oidc_issuer_url" {
  default = "https://app.terraform.io"
}

variable "oidc_audience" {
  default = "kubernetes"
}

variable "oidc_idp_name" {
  default = "terraform-cloud"
}

variable "rbac_group_oidc_claim" {
  default = "terraform_organization_name"
}

variable "rbac_oidc_group_name" {
  type = string
}

variable "rbac_group_cluster_role" {
  default = "cluster-admin"
}

resource "aws_eks_identity_provider_config" "oidc_config" {
  cluster_name = var.cluster_name

  oidc {
    identity_provider_config_name = var.oidc_idp_name
    client_id                     = var.oidc_audience
    issuer_url                    = var.oidc_issuer_url
    username_claim                = "sub"
    groups_claim                  = var.rbac_group_oidc_claim
  }
}

data "aws_eks_cluster" "target_eks" {
  name = var.cluster_name
}

data "aws_eks_cluster_auth" "target_eks_auth" {
  name = var.cluster_name
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.target_eks.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.target_eks.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.target_eks_auth.token
}

resource "kubernetes_cluster_role_binding_v1" "oidc_role" {
  metadata {
    name = "odic-identity"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = var.rbac_group_cluster_role
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Group"
    name      = var.rbac_oidc_group_name
  }
}
