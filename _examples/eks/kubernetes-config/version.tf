terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
      version = "~> 2.1.0"
    }
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.39.0"
    }
  }
}
