# Step 1 — SDKv2 provider (last published release before the Framework migration).
# This is the version a real user's terraform.tfstate will have been written by.
# Pin to exactly 3.2.1 so the registry always serves the same binary.
# Do NOT activate dev_overrides for this step.

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "3.2.1"
    }
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "kind-kind"
}
