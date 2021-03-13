terraform {
  required_providers {
    kubernetes = {
      source = "localhost/test/kubernetes"
      version = "9.9.9"
    }
    helm = {
      source  = "localhost/test/helm"
      version = "9.9.9"
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
# a longer time than the token TTL. The downside of this approach is that the binary must be present
# on the system running terraform, either in $PATH as shown below, or in another location, which can be
# specified in the `command`.
# See the commented provider blocks below for alternative configuration options.
provider "kubernetes" {
  host                   = data.aws_eks_cluster.default.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", module.vpc.cluster_name]
    command     = "aws"
  }
}

# This configuration is also valid, but the token may expire during long-running applies.
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
      args        = ["eks", "get-token", "--cluster-name", module.vpc.cluster_name]
      command     = "aws"
    }
  }
}

provider "aws" {
  region = var.region
}

module "vpc" {
  source = "./vpc"
}

module "cluster" {
  source  = "terraform-aws-modules/eks/aws"
  version = "14.0.0"

  vpc_id  = module.vpc.vpc_id
  subnets = module.vpc.subnets

  cluster_name    = module.vpc.cluster_name
  cluster_version = var.kubernetes_version
  manage_aws_auth = false # Managed in ./kubernetes-config/main.tf instead.
  # This kubeconfig expires in 15 minutes, so we'll use an exec block instead.
  # See ./kubernetes-config/main.tf provider block for details.
  write_kubeconfig = false

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
  k8s_node_role_arn = module.cluster.worker_iam_role_arn
}
