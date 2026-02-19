# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

resource "tls_private_key" "example" {
  algorithm = "ECDSA"
  rsa_bits  = "4096"
}

resource "tls_cert_request" "example" {
  key_algorithm   = "ECDSA"
  private_key_pem = tls_private_key.example.private_key_pem

  subject {
    common_name  = var.example_user
    organization = var.example_org
  }
}

resource "kubernetes_certificate_signing_request" "example" {
  metadata {
    name = "example"
  }
  spec {
    request = tls_cert_request.example.cert_request_pem
    usages  = ["client auth", "server auth"]
  }
  auto_approve = true
}

resource "kubernetes_secret" "example" {
  metadata {
    name = "test-secret"
  }
  data = {
    "tls.crt" = kubernetes_certificate_signing_request.example.certificate
    "tls.key" = tls_private_key.example.private_key_pem
  }
  type = "kubernetes.io/tls"
}

resource "kubernetes_pod" "main" {
  metadata {
    name = "test-pod"
  }
  spec {
    container {
      name    = "default"
      image   = "alpine:latest"
      command = ["cat", "/etc/test/tls.crt"]
      volume_mount {
        mount_path = "/etc/test"
        name       = "secretvol"
      }
    }
    volume {
      name = "secretvol"
      secret {
        secret_name = kubernetes_secret.example.metadata[0].name
      }
    }
  }
}
