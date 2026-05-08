# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "random_string" "rand" {
  length  = 8
  lower   = true
  special = false
}

locals {
  cluster_name    = var.cluster_name != "" ? var.cluster_name : "test-cluster-${random_string.rand.result}"
  cidr            = "10.0.0.0/16"
  az_count        = min(var.az_span, length(data.aws_availability_zones.available.names))
  azs             = slice(data.aws_availability_zones.available.names, 0, local.az_count)
  private_subnets = [for i, z in local.azs : cidrsubnet(local.cidr, 8, i)]
  public_subnets  = [for i, z in local.azs : cidrsubnet(local.cidr, 8, i + local.az_count)]
  node_count      = var.nodes_per_az * local.az_count

  tags = {
    team        = "terraform-kubernetes-providers"
    environment = "test"
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.15"

  cluster_name                    = local.cluster_name
  cluster_version                 = var.cluster_version
  cluster_endpoint_private_access = true
  cluster_endpoint_public_access  = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  eks_managed_node_groups = {
    default_node_group = {
      ami_type                   = "AL2_ARM_64"
      capacity_type              = var.capacity_type
      desired_size               = local.node_count
      min_size                   = 1
      max_size                   = local.node_count
      instance_types             = [var.instance_type]
      use_custom_launch_template = false

      iam_role_additional_policies = {
        AmazonEBSCSIDriverPolicy = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
      }
    }

  }

  tags = local.tags
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = local.cluster_name
  cidr = local.cidr

  azs             = local.azs
  private_subnets = local.private_subnets
  public_subnets  = local.public_subnets

  create_egress_only_igw = true

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
        exec = {
          apiVersion = "client.authentication.k8s.io/v1"
          command    = "aws"
          args = [
            "eks", "get-token", "--cluster-name", local.cluster_name
          ]
        }
      }
    }]
  })
}

resource "local_file" "kubeconfig" {
  content  = local.kubeconfig
  filename = "${path.module}/kubeconfig"
}
