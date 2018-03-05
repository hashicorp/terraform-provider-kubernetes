variable "kubernetes_version" {
  type = "string"
}

variable "zone_name" {
  type = "string"
}

variable "bucket_prefix" {
  type = "string"
  default = "kops-tfacc"
}

variable "private_ssh_key_filename" {
  type = "string"
  default = "id_rsa"
}

resource "random_id" "name" {
  byte_length = 2
}

locals {
  cluster_name = "k.${random_id.name.hex}.${var.zone_name}"
  bucket_name = "${var.bucket_prefix}-${random_id.name.hex}"
  public_ssh_key_location = "${path.module}/${var.private_ssh_key_filename}.pub"
  tmp_creds_location = "${path.module}/google-creds.json"
}

data "google_compute_zones" "available" {}

data "http" "ipinfo" {
  url = "http://ipinfo.io/ip"
}

resource "null_resource" "kops" {
  provisioner "local-exec" {
    command = <<EOF
ssh-keygen -P "" -t rsa -f ./${var.private_ssh_key_filename}
export TMP_CREDS_PATH=${local.tmp_creds_location}
export CLUSTER_NAME=${local.cluster_name}
export BUCKET_NAME=${local.bucket_name}
export KUBERNETES_VERSION=${var.kubernetes_version}
export IP_ADDRESS=${chomp(data.http.ipinfo.body)}
export ZONES=${data.google_compute_zones.available.names[0]}
export SSH_PUBKEY_PATH=${local.public_ssh_key_location}
./kops-create.sh
EOF
  }

  provisioner "local-exec" {
    when = "destroy"
    command = <<EOF
export TMP_CREDS_PATH=${local.tmp_creds_location}
export CLUSTER_NAME=${local.cluster_name}
export BUCKET_NAME=${local.bucket_name}
./kops-delete.sh
rm -f ${local.tmp_creds_location}
EOF
  }
}

output "cluster_name" {
  value = "${local.cluster_name}"
}

output "google_zone" {
  value = "${data.google_compute_zones.available.names[0]}"
}
