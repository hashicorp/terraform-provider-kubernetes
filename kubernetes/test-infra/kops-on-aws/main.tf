provider "aws" { }

variable "route53_zone" {
  type = "string"
}

variable "kubernetes_version" {
  type = "string"
}

variable "s3_bucket_prefix" {
  type = "string"
  default = "kops-tfacc"
}

variable "private_ssh_key_filename" {
  type = "string"
  default = "id_rsa"
}

resource "random_id" "name" {
  byte_length = 8
}

locals {
  cluster_name = "${random_id.name.hex}.kops.${var.route53_zone}"
  bucket_name = "${var.s3_bucket_prefix}-${random_id.name.hex}"
  public_ssh_key_location = "${path.module}/${var.private_ssh_key_filename}.pub"
}

data "aws_availability_zones" "available" {}

data "http" "ipinfo" {
  url = "http://ipinfo.io/ip"
}

resource "tls_private_key" "ssh" {
  algorithm   = "RSA"
}

resource "null_resource" "kops" {
  provisioner "local-exec" {
    command = <<EOF
ssh-keygen -P "" -t rsa -f ./${var.private_ssh_key_filename}
export CLUSTER_NAME=${local.cluster_name}
export BUCKET_NAME=${local.bucket_name}
export KUBERNETES_VERSION=${var.kubernetes_version}
export IP_ADDRESS=${chomp(data.http.ipinfo.body)}
export ZONES=${data.aws_availability_zones.available.names[0]}
export SSH_PUBKEY_PATH=${local.public_ssh_key_location}
./kops-create.sh
EOF
  }

  provisioner "local-exec" {
    when = "destroy"
    command = <<EOF
export CLUSTER_NAME=${local.cluster_name}
export BUCKET_NAME=${local.bucket_name}
./kops-delete.sh
EOF
  }
}

output "cluster_name" {
  value = "${local.cluster_name}"
}

output "availability_zone" {
  value = "${data.aws_availability_zones.available.names[0]}"
}
