---
subcategory: "certificates/v1"
page_title: "Kubernetes: kubernetes_certificate_signing_request_v1"
description: |-
  Use this resource to generate TLS certificates using Kubernetes.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_certificate_signing_request_v1" "example" {
  metadata {
    name = "example"
  }
  spec {
    usages      = ["client auth", "server auth"]
    signer_name = "kubernetes.io/kube-apiserver-client"

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
    "tls.crt" = kubernetes_certificate_signing_request_v1.example.certificate
    "tls.key" = tls_private_key.example.private_key_pem # key used to generate Certificate Request
  }
  type = "kubernetes.io/tls"
}
```

## Generating a New Certificate

Since the certificate is a logical resource that lives only in the Terraform state, it will persist until it is explicitly destroyed by the user.

In order to force the generation of a new certificate within an existing state, the certificate instance can be "tainted":

```
terraform taint kubernetes_certificate_signing_request_v1.example
```

A new certificate will then be generated on the next `terraform apply`.
