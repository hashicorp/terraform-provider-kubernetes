---
subcategory: "certificates/v1beta1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_certificate_signing_request"
description: |-
  Use this resource to generate TLS certificates using Kubernetes.
---

# kubernetes_certificate_signing_request

Use this resource to generate TLS certificates using Kubernetes.

This is a *logical resource*, so it contributes only to the current Terraform state and does not persist any external managed resources.

This resource enables automation of [X.509](https://www.itu.int/rec/T-REC-X.509) credential provisioning (including TLS/SSL certificates). It does this by creating a CertificateSigningRequest using the Kubernetes API, which generates a certificate from the Certificate Authority (CA) configured in the Kubernetes cluster. The CSR can be approved automatically by Terraform, or it can be approved by a custom controller running in Kubernetes. See [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/) for all available options pertaining to CertificateSigningRequests.

## Example Usage

```hcl
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
```

## Argument Reference

The following arguments are supported:

* `auto_approve` - (Optional) Automatically approve the CertificateSigningRequest. Defaults to 'true'.
* `metadata` - (Required) Standard certificate signing request's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the specification of the desired behavior of the deployment. For more info see [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the certificate signing request that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the certificate signing request. May match selectors of replication controllers and services.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the certificate signing request, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

#### Attributes

* `certificate` - The signed certificate PEM data.
* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this certificate signing request that can be used by clients to determine when certificate signing request has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this certificate signing request. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `request` - (Required) Base64-encoded PKCS#10 CSR data.
* `signer_name` - (Optional) Requested signer for the request. It is a qualified name in the form: `scope-hostname.io/name`. If empty, it will be defaulted: 1. If it's a kubelet client certificate, it is assigned "kubernetes.io/kube-apiserver-client-kubelet". 2. If it's a kubelet serving certificate, it is assigned "kubernetes.io/kubelet-serving". 3. Otherwise, it is assigned "kubernetes.io/legacy-unknown". Distribution of trust for signers happens out of band.
* `usages` - (Required) Specifies a set of usage contexts the key will be valid for. See https://godoc.org/k8s.io/api/certificates/v1beta1#KeyUsage

## Generating a New Certificate

Since the certificate is a logical resource that lives only in the Terraform state,
it will persist until it is explicitly destroyed by the user.

In order to force the generation of a new certificate within an existing state, the
certificate instance can be "tainted":

```
terraform taint kubernetes_certificate_signing_request.example
```

A new certificate will then be generated on the next ``terraform apply``.
