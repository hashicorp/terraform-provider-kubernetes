# CertificateSigningRequest example

This example creates a CertificateSigningRequest in Kubernetes and uses the resulting certificate in a pod. To run this example, have a Kubernetes cluster available (or create one for testing using [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)). Ensure your [kubeconfig](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) exists in the default location, or specify the `KUBECONFIG` environment variable. Then apply the Terraform configs located in this example directory:

```
cd _examples/certificate-signing-request/
terraform init
terraform apply
```

The resulting resources can be viewed using kubectl.

```
$ kubectl logs test-pod
-----BEGIN CERTIFICATE-----
MIICIjCCAQqgAwIBAgIQAz7g4BfKx1rHJ8zRceMgsTANBgkqhkiG9w0BAQsFADAV
MRMwEQYDVQQDEwptaW5pa3ViZUNBMB4XDTIwMDcyNTAwMDkyOFoXDTIxMDcyNTAw
MDkyOFowKjEYMBYGA1UEChMPZXhhbXBsZSBjbHVzdGVyMQ4wDAYDVQQDEwVhZG1p
bjBOMBAGByqGSM49AgEGBSuBBAAhAzoABGrufaGO4MMBleMKXVmcDEOknmqG/2A2
HbBISW1Y1bQTv9JF72ZzXclNglwDTpgSjL7HXRCY0JOgoy8wLTAdBgNVHSUEFjAU
BggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADANBgkqhkiG9w0BAQsF
AAOCAQEACnJMtZMb2abgqZVeWoPSfFNed2QFSm/+7i4T/L0wKFR4I/XLRIvfhh9z
kLRf7Gok4w4Og3GQnUSOARZGLFmZaqqRXmbQyTbUWW5XHeH5HkdKIPwwdAmgCo6L
azwfHRkeLMOJaB3WDgAL4y1Mn42FYwlnMAsiuOydKFfV4BQGNEeuP+dFtH2wAgDq
7uRi3OMvPuHooO3b3oEKWVM9A5yODLNbAhTsBJL8cmFnUXeCqEKvGkNSQDtb9Kw/
8UGdRuzlwhvD/LKgF57LvGldijQgP/4lFjalnkSkXDwtoscXInwzXgV9dx7a+syn
HsD/cyWXbvABKXn5fg5rDtGMUgdIWQ==
-----END CERTIFICATE-----
```
