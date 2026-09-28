terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "99.0.0"
    }
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "kind-kind"
}
