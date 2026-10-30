terraform {
  required_version = ">= 1.12.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.37.1"
    }
  }
}

variable "config_path" {
  type    = string
  default = "~/.kube/config"
}

variable "config_context" {
  type    = string
  default = "kind-kind"
}

variable "namespace" {
  type    = string
  default = "default"
}

variable "run_id" {
  type    = string
  default = "manual"
}

variable "phase" {
  type    = string
  default = "before"

  validation {
    condition     = contains(["before", "after"], var.phase)
    error_message = "phase must be before or after."
  }
}

provider "kubernetes" {
  config_path        = pathexpand(var.config_path)
  config_context     = var.config_context
  ignore_annotations = ["^outside[.]example/keep$"]
  ignore_labels      = ["^outside[.]example/owner$"]
}


moved {
  from = kubernetes_role.test
  to   = kubernetes_role_v1.test
}

resource "kubernetes_role_v1" "test" {
  metadata {
    name      = "role-sanity-pre-unversioned-${var.run_id}"
    namespace = var.namespace
    annotations = {
      managed = var.phase
    }
    labels = var.phase == "before" ? {
      managed = "before"
      remove  = "yes"
      } : {
      managed = "after"
      added   = "yes"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "configmaps"]
    verbs      = var.phase == "before" ? ["get", "list"] : ["get", "list", "watch"]
  }

  rule {
    api_groups     = ["apps"]
    resources      = ["deployments"]
    resource_names = ["example"]
    verbs          = ["get"]
  }
}

output "role_id" {
  value = kubernetes_role_v1.test.id
}

output "role_name" {
  value = kubernetes_role_v1.test.metadata[0].name
}

output "role_uid" {
  value = kubernetes_role_v1.test.metadata[0].uid
}