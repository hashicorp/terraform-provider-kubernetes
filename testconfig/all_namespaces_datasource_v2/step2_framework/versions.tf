# Step 2 — Local Framework binary (dev_overrides active).
# The version constraint is intentionally loose; dev_overrides bypasses the
# registry entirely and the constraint is only used for lockfile metadata.
# Terraform will print a warning that dev_overrides are in effect — that is
# expected and required to prove the local binary is running.

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 3.0.0"
    }
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "kind-kind"
}
