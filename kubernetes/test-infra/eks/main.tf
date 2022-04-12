resource "random_string" "rand" {
  length  = 8
  lower   = true
  special = false
}

locals {
  cluster_name    = "test-cluster-${random_string.rand.result}"
  cluster_version = var.cluster_version
  region          = var.region

  tags = {
    team        = "terraform-kubernetes-providers"
    environment = "test"
  }
}

provider "aws" {
  region = local.region
}

module "eks" {
  source = "terraform-aws-modules/eks/aws"
  version = "~> 18.11"

  cluster_name                    = local.cluster_name
  cluster_version                 = local.cluster_version
  cluster_endpoint_private_access = true
  cluster_endpoint_public_access  = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  eks_managed_node_group_defaults = {
    instance_types = [var.instance_type]
    min_size       = 1
    max_size       = var.node_count
    desired_size   = var.node_count
  }

  eks_managed_node_groups = {
    default_node_group = {
      create_launch_template = false
      launch_template_name   = ""
    }
  }

  tags = local.tags
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 3.0"

  name = local.cluster_name
  cidr = "10.0.0.0/16"

  azs             = ["${local.region}a", "${local.region}b"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.4.0/24", "10.0.5.0/24"]

  create_egress_only_igw  = true

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true
  enable_dns_support   = true

  public_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                      = 1
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"             = 1
  }

  tags = local.tags
}

data "aws_eks_cluster_auth" "this" {
  name = module.eks.cluster_id
}

locals {
  kubeconfig = yamlencode({
    apiVersion      = "v1"
    kind            = "Config"
    current-context = "terraform"
    clusters = [{
      name = module.eks.cluster_id
      cluster = {
        certificate-authority-data = module.eks.cluster_certificate_authority_data
        server                     = module.eks.cluster_endpoint
      }
    }]
    contexts = [{
      name = "terraform"
      context = {
        cluster = module.eks.cluster_id
        user    = "terraform"
      }
    }]
    users = [{
      name = "terraform"
      user = {
        token = data.aws_eks_cluster_auth.this.token
      }
      user = {
        exec = {
          apiVersion = "client.authentication.k8s.io/v1alpha1"
          command = "aws"
          args = [
            "eks", "get-token", "--cluster-name", local.cluster_name
          ]
        }
      }
    }]
  })
}

resource "local_file" "kubeconfig" {
  content = local.kubeconfig
  filename = "${path.module}/kubeconfig"
}