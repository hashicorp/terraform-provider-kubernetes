# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_certificate_signing_request" "example" {
  metadata {
    name = "example"
  }
  spec {
    usages  = ["client auth", "server auth"]
    request = <<EOT
-----BEGIN CERTIFICATE REQUEST-----
MIHSMIGBAgEAMCoxGDAWBgNVBAoTD2V4YW1wbGUgY2x1c3RlcjEOMAwGA1UEAxMF
YWRtaW4wTjAQBgcqhkjOPQIBBgUrgQQAIQM6AASSG8S2+hQvfMq5ucngPCzK0m0C
ImigHcF787djpF2QDbz3oQ3QsM/I7ftdjB/HHlG2a5YpqjzT0KAAMAoGCCqGSM49
BAMCA0AAMD0CHQDErNLjX86BVfOsYh/A4zmjmGknZpc2u6/coTHqAhxcR41hEU1I
DpNPvh30e0Js8/DYn2YUfu/pQU19
-----END CERTIFICATE REQUEST-----
EOT
  }
  auto_approve = true
}


resource "kubernetes_secret" "example" {
  metadata {
    name = "example"
  }
  data = {
    "tls.crt" = kubernetes_certificate_signing_request.example.certificate
    "tls.key" = tls_private_key.example.private_key_pem # key used to generate Certificate Request
  }
  type = "kubernetes.io/tls"
}
