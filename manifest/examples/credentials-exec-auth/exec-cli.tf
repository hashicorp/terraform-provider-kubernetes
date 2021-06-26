# This examples demonstrate how to configure the provider for accessing an EKS cluster using credentials dispatched by the AWS cli.
#
# It is based on the instructions described here: https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html#create-kubeconfig-manually
#
# Requirement: the AWS cli tool should be installed and confirmed working.
# If it is not accesible in the PATH, the full path to the 'aws' tool should be used in the "command" attribute.

variable "cluster_ca" {
  type = string
}

variable "cluster_region" {
  type    = string
  default = "eu-central-1"
}

variable "cluster_endpoint" {
  type = string
}

variable "cluster_name" {
  type = string
}

provider "kubernetes-alpha" {
  host = var.cluster_endpoint

  # Cluster CA certificate obtained from EKS
  cluster_ca_certificate = file(var.cluster_ca)

  exec = {
    api_version = "client.authentication.k8s.io/v1alpha1"

    command = "aws" # this is the actual 'aws' cli tool

    args = ["--region", var.cluster_region, "eks", "get-token", "--cluster-name", var.cluster_name]

    env = {
      # Credentials for the AWS cli tool
      "AWS_PROFILE" = "hashicorp"

      # Alternatively, set an access key ID and secret key
      #
      # "AWS_ACCESS_KEY_ID"     = "my-access-key-id"
      # "AWS_SECRET_ACCESS_KEY" = "my-secret-key-data"
    }
  }
}

resource "kubernetes_manifest" "test-namespace" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Namespace"
    "metadata" = {
      "name" = "tf-demo"
    }
  }
}
