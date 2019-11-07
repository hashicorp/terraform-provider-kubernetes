provider "google" {
  version = "~> 2.19"

  // Provider settings to be provided via ENV variables
  region  = var.gcp_region
  project = var.gcp_project
}

provider "kubernetes" {
  version = "~> 1.9"
}
