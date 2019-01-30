module "cluster" {
  source = "./cluster"

  kubernetes_version = "${var.kubernetes_version}"
  region             = "${var.region}"
  workers_count      = "${var.workers_count}"
  workers_type       = "${var.workers_type}"
}

module "node-config" {
  source = "./node-config"

  k8s_node_role_arn = "${module.cluster.k8s_node_role_arn}"
  kubeconfig_path   = "${module.cluster.kubeconfig_path}"
}
