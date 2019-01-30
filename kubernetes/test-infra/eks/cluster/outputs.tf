#
# Outputs
#

output "kubeconfig_path" {
  value = "${local_file.kubeconfig.filename}"
}

output "k8s_node_role_arn" {
  value = "${aws_iam_role.k8s-acc-node.arn}"
}
