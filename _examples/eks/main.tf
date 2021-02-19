terraform {
  required_providers {
   # NOTE:
   # Although we're specifying this version of the kubernetes provider in the root module,
   # it is the responsibility of each sub-module to specify their respective provider versions.
   # It is possible that a different version will be used in the EKS module, which is called in a later step in this file.
    kubernetes = {
      source = "hashicorp/kubernetes"
      version = ">= 2.0.3"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.1.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "3.22.0"
    }
  }
}

data "aws_eks_cluster" "default" {
  name = module.cluster.cluster_id
}

# This configuration relies on a plugin binary to fetch the token to the EKS cluster.
# The main advantage is that the token will always be up-to-date, even when the `terraform apply` runs for
# a longer time than the token TTL. It will also protect against an issue in Terraform 0.14 where the token
# data source is not refreshed during destroy.
# https://github.com/hashicorp/terraform/issues/28179
#
# The downside of this approach is that the binary must be present on the system running terraform,
# either in $PATH as shown below, or in another location, which can be specified in the `command`.
# See the commented provider blocks below for alternative configuration options.
provider "kubernetes" {
  host                   = data.aws_eks_cluster.default.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["token", "--cluster-id", module.vpc.cluster_name]
    command     = "aws-iam-authenticator"
  }
}

# This configuration is also valid, but users may prefer not to install the full aws binary onto CI systems.
#provider "kubernetes" {
#  host                   = data.aws_eks_cluster.default.endpoint
#  cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
#  exec {
#    api_version = "client.authentication.k8s.io/v1alpha1"
#    args        = ["eks", "get-token", "--cluster-name", module.vpc.cluster_name]
#    command     = "aws"
#  }
#}

# This configuration is also valid, but the token may expire during long-running applies.
# The kubernetes resources could also fail to delete, if running on Terraform 0.14.x.
# https://github.com/hashicorp/terraform/issues/28179
# data "aws_eks_cluster_auth" "default" {
#  name = module.cluster.cluster_id
#}
#provider "kubernetes" {
#  host                   = data.aws_eks_cluster.default.endpoint
#  cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
#  token                  = data.aws_eks_cluster_auth.default.token
#}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.default.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
    exec {
      api_version = "client.authentication.k8s.io/v1alpha1"
      args        = ["token", "--cluster-id", module.vpc.cluster_name]
      command     = "aws-iam-authenticator"
    }
  }
}

module "vpc" {
  source = "./vpc"
}

module "cluster" {
  source  = "terraform-aws-modules/eks/aws"
  version = "14.0.0"

  vpc_id  = module.vpc.vpc_id
  subnets = module.vpc.subnets

  cluster_name     = module.vpc.cluster_name
  cluster_version  = var.kubernetes_version
  manage_aws_auth  = true
  write_kubeconfig = true

  workers_group_defaults = {
    root_volume_type = "gp2"
  }
  worker_groups = [
    {
      instance_type        = var.workers_type
      asg_desired_capacity = var.workers_count
      asg_max_size         = 4
    },
  ]

  tags = {
    environment = "test"
  }
}

module "kubernetes-config" {
  cluster_name      = module.cluster.cluster_id # creates dependency on cluster creation
  source            = "./kubernetes-config"
}
