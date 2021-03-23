terraform {
  required_providers {
    # This is the locally compiled version of the provider, based on the current branch.
    kubernetes-local = {
      source = "localhost/test/kubernetes"
      version = "9.9.9"
    }
    # The following block configures the latest released version of the provider, which is needed for the EKS cluster module.
    # This configuration is a work-around, because required_providers blocks are not inherited by sub-modules.
    # A "required_providers" block needs to be added to all sub-modules in order to use a custom "source" and "version".
    # Otherwise, the sub-module will use defaults, which in our case means an empty provider config.
    # https://github.com/hashicorp/terraform/issues/27361
    kubernetes-released = {
      source = "hashicorp/kubernetes"
      version = ">= 2.0.2"
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
provider "kubernetes-released" {
  host                   = data.aws_eks_cluster.default.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", module.vpc.cluster_name]
    command     = "aws"
  }
}

# This tests a progressive apply scenario where the kubeconfig is created in the same apply as Kubernetes resources.
# It should alert us to issues like this one before they're released.
# https://github.com/hashicorp/terraform-provider-kubernetes/issues/1142
provider "kubernetes-local" {
  config_path = module.cluster.kubeconfig_filename
}

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
}

module "vpc" {
  source = "./vpc"
}

module "cluster" {
  providers         =  {kubernetes = kubernetes-released}
  source  = "terraform-aws-modules/eks/aws"
  version = "14.0.0"

  vpc_id  = module.vpc.vpc_id
  subnets = module.vpc.subnets

  cluster_name     = module.vpc.cluster_name
  cluster_version  = var.kubernetes_version
  manage_aws_auth  = true
  write_kubeconfig = true
  kubeconfig_name  = "kubeconfig"

  # See this file for more options
  # https://github.com/terraform-aws-modules/terraform-aws-eks/blob/master/local.tf#L28
  workers_group_defaults = {
    root_volume_type = "gp2"
  }

  worker_groups = [
    {
      name                 = module.vpc.cluster_name
      instance_type        = "m4.large"
      asg_min_size         = 1
      asg_max_size         = 4
      asg_desired_capacity = 2
    },
  ]

  tags = {
    environment = "test"
  }
}

module "kubernetes-config" {
  providers         =  {kubernetes = kubernetes-local}
  cluster_name      = module.cluster.cluster_id # creates dependency on cluster creation
  source            = "./kubernetes-config"
  k8s_node_role_arn = module.cluster.worker_iam_role_arn
}
