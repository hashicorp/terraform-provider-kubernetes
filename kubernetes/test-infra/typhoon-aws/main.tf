# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals {
  kubeconfig_path = "${path.root}/kubeconfig"
}

resource "tls_private_key" "typhoon-acc" {
  algorithm = "RSA"
}

resource "local_file" "public_key_openssh" {
  content  = tls_private_key.typhoon-acc.public_key_openssh
  filename = "${path.cwd}/${var.cluster_name}.pub"
}

resource "local_file" "private_key_pem" {
  content  = tls_private_key.typhoon-acc.private_key_pem
  filename = "${path.cwd}/${var.cluster_name}"
}

resource "null_resource" "ssh-key" {
  provisioner "local-exec" {
    command     = format("chmod 600 %v", local_file.private_key_pem.filename)
    working_dir = path.cwd
  }
  provisioner "local-exec" {
    command     = format("ssh-add %v", local_file.private_key_pem.filename)
    working_dir = path.cwd
  }
}

data "aws_route53_zone" "typhoon-acc" {
  name = var.base_domain
}

output "kubeconfig_path" {
  value = local.kubeconfig_path
}