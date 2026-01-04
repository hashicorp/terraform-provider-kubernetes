# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.0"
    }
  }
}

variable "kube_config_file" {
  default = "~/.kube/config"
}

provider "kubernetes" {
  config_path = var.kube_config_file
}

resource "kubernetes_job" "test" {
  metadata {
    name = "test"
  }
  spec {
    active_deadline_seconds = 120
    backoff_limit           = 10
    completions             = 10
    parallelism             = 2
    template {
      metadata {}
      spec {
        container {
          name    = "hello"
          image   = "busybox"
          command = ["sleep", "30"]
        }
      }
    }
  }
  wait_for_completion = true
}
