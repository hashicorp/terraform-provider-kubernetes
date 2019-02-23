module "vpc" {
  source = "./vpc"
}

module "cluster" {
  source  = "terraform-aws-modules/eks/aws"
  version = "2.2.0"

  vpc_id  = "${module.vpc.vpc_id}"
  subnets = ["${module.vpc.subnets}"]

  cluster_name    = "${module.vpc.cluster_name}"
  cluster_version = "${var.kubernetes_version}"

  worker_groups = [
    {
      instance_type        = "${var.workers_type}"
      asg_desired_capacity = "${var.workers_count}"
      asg_max_size         = "10"
    },
  ]

  write_kubeconfig   = true
  config_output_path = "${local.kubeconfig_path}/"
  manage_aws_auth    = false

  tags = {
    environment = "test"
  }
}

module "node-config" {
  source = "./node-config"

  k8s_node_role_arn = ["${list(module.cluster.worker_iam_role_arn)}"]
  cluster_ca        = "${module.cluster.cluster_certificate_authority_data}"
  cluster_endpoint  = "${module.cluster.cluster_endpoint}"
  kubeconfig        = "${module.cluster.kubeconfig}"
}
