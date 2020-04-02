resource "tls_private_key" "typhoon-acc" {
  algorithm = "RSA"
}

resource "local_file" "public_key_openssh" {
  content    = tls_private_key.typhoon-acc.public_key_openssh
  filename   = "${path.cwd}/${var.cluster_name}.pub"
}

resource "local_file" "private_key_pem" {
  content    = tls_private_key.typhoon-acc.private_key_pem
  filename   = "${path.cwd}/${var.cluster_name}"
}

resource "null_resource" "ssh-key" {
  provisioner "local-exec" {
    command = format("chmod 600 %v", local_file.private_key_pem.filename)
    working_dir = path.cwd
  }
  provisioner "local-exec" {
    command = format("ssh-add %v", local_file.private_key_pem.filename)
    working_dir = path.cwd
  }
}

data "aws_route53_zone" "typhoon-acc" {
  name = var.base_domain
}

module "typhoon-acc" {
  source = "git::https://github.com/poseidon/typhoon//aws/fedora-coreos/kubernetes?ref=v1.18.0" # set the desired Kubernetes version here

  cluster_name = var.cluster_name
  dns_zone     = var.base_domain
  dns_zone_id  = data.aws_route53_zone.typhoon-acc.zone_id

  # node configuration
  ssh_authorized_key = tls_private_key.typhoon-acc.public_key_openssh

  worker_count = var.worker_count
  controller_count = var.controller_count
  worker_type  = var.controller_type
  controller_type = var.worker_type
}

resource "local_file" "typhoon-acc" {
  content  = module.typhoon-acc.kubeconfig-admin
  filename = "kubeconfig"
}

output "kubeconfig_path" {
  value = "${path.cwd}/${local_file.typhoon-acc.filename}"
}
