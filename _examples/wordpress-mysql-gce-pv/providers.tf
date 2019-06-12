provider "google" {
  region  = var.gcp_region // Provider settings to be provided via ENV variables
  version = "~> 2.8"
}

provider "kubernetes" {
  version = "~> 1.7"
}
