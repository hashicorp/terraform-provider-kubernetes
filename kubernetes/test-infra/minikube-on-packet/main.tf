provider "packet" { }

variable "kubernetes_version" {
  type = "string"
  description = "See 'minikube get-k8s-versions' for all available versions"
}

variable "packet_facility" {
  type = "string"
  description = "See https://www.packet.net/developers/api/facilities/ for all available facilities"
  default = "ams1"
}

variable "packet_plan" {
  type = "string"
  description = "See https://www.packet.net/developers/api/plans/ for all available plans"
  default = "baremetal_1"
}

variable "local_tunnel_port" {
  type = "string"
  default = "32000"
}

variable "kubernetes_api_port" {
  type = "string"
  default = "8443"
}

variable "private_ssh_key_location" {
  type = "string"
  default = "./id_ecdsa"
}

variable "dotminikube_path" {
  type = "string"
  default = "client/.minikube"
}

resource "random_id" "name" {
  byte_length = 8
}

resource "tls_private_key" "ssh" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "null_resource" "ssh_key" {
  provisioner "local-exec" {
    command = "echo \"${tls_private_key.ssh.private_key_pem}\" > ${var.private_ssh_key_location} && chmod 600 ${var.private_ssh_key_location}"
  }
}

resource "packet_project" "main" {
  name = "minikube-test-${random_id.name.hex}"
}

resource "packet_ssh_key" "default" {
  name       = "default"
  public_key = "${tls_private_key.ssh.public_key_openssh}"
}

resource "packet_device" "minikube" {
  hostname         = "minikube"
  plan             = "${var.packet_plan}"
  facility         = "${var.packet_facility}"
  operating_system = "centos_7"
  billing_cycle    = "hourly"
  project_id       = "${packet_project.main.id}"

  provisioner "file" {
    connection {
      type        = "ssh"
      user        = "root"
      private_key = "${tls_private_key.ssh.private_key_pem}"
    }
    source      = "${path.module}/10-install-virtualbox.sh"
    destination = "/tmp/10-install-virtualbox.sh"
  }
  provisioner "file" {
    connection {
      type        = "ssh"
      user        = "root"
      private_key = "${tls_private_key.ssh.private_key_pem}"
    }
    source      = "${path.module}/20-install-minikube.sh"
    destination = "/tmp/20-install-minikube.sh"
  }

  provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "root"
      private_key = "${tls_private_key.ssh.private_key_pem}"
    }
    inline = [
      "chmod a+x /tmp/10-install-virtualbox.sh && chmod a+x /tmp/20-install-minikube.sh",
      "/tmp/10-install-virtualbox.sh | tee /var/log/provisioning-10-virtualbox.log",
      "/tmp/20-install-minikube.sh | tee /var/log/provisioning-20-minikube.log",
      "minikube start --kubernetes-version=v${var.kubernetes_version}",
      # Extract certs so they can be transfered back to client
      "mkdir -p /tmp/${var.dotminikube_path}",
      "minikube ip | tr -d \"\n\" > /tmp/client/local-ip.txt",
      "cp -r ~/.minikube/{ca.crt,client.crt,client.key} /tmp/${var.dotminikube_path}/",
    ]
  }

  # Pull certs & local IP so we can connect to minikube
  provisioner "local-exec" {
    command = "scp -i ${var.private_ssh_key_location} -r -o StrictHostKeyChecking=no root@${self.access_public_ipv4}:/tmp/client ./"
  }

  depends_on = ["packet_ssh_key.default"]
}

# Not a great way to setup an SSH tunnel, but it's the only reasonable one
# until https://github.com/hashicorp/terraform/issues/8367 is a thing
resource "null_resource" "ssh_tunnel" {
  provisioner "local-exec" {
    command = <<EOF
ssh -i ${var.private_ssh_key_location} -o StrictHostKeyChecking=no -o ControlMaster=no -M -S tunnel-ctrl.socket -fNnT \
  -L ${var.local_tunnel_port}:$(cat ./client/local-ip.txt):${var.kubernetes_api_port} \
  root@${packet_device.minikube.access_public_ipv4} >./tunnel.stdout.log 2>./tunnel.stderr.log
EOF
  }
  provisioner "local-exec" {
    when = "destroy"
    command = "ssh -i ${var.private_ssh_key_location} -S tunnel-ctrl.socket -O exit root@${packet_device.minikube.access_public_ipv4}"
  }
}

output "ip_address" {
  value = "${packet_device.minikube.access_public_ipv4}"
}

output "local_tunnel_port" {
  value = "${var.local_tunnel_port}"
}

output "dotminikube_path" {
  value = "${var.dotminikube_path}"
}
