provider "google" {
  region  = var.region // Provider settings to be provided via ENV variables
  version = "~> 2.8"
}

provider "kubernetes" {
  version = "~> 1.7"
}
