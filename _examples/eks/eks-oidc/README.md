# Configuring EKS for OIDC identity providers

Kubernetes, and by extension EKS, natively supports OIDC as an indentity provider to which it will delegate user authentication. The result of a successful authentication through OIDC is a base64 encoded token of data describing the user identity. The format of this token is called JWT (JSON Web Token) and is described by [RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519).

The Kubernetes and Helm providers for Terraform are already designed to accept JWTs as indentity carriers. A token is passed to the provider by setting the `token` attribute on the provider block (or the `KUBE_TOKEN` environment variable).

Terraform Cloud can act as an OIDC identity provider to kubernetes, issuing JWT tokens that it designates as ["workload identity"](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/workload-identity-tokens). This module is designed around using TFC as an indentity provider, but will likely work with  any OIDC compliant IDp, such as Okta.

# OIDC on EKS

EKS can be configured with an external IDp through the [`aws_eks_identity_provider_config`](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/eks_identity_provider_config) Terraform resource. This module is a thin wrapper around it, adding some meaningful defaults in the context of TFC (which can, of course be overridden) as well as the necessary RBAC role binding to grant permissions to the indentity obtained from OIDC.

To make use of the module, roughly follow the following steps (adapt for you actual needs):

1. Create an EKS cluster

   Use the method of your choice to spin up an EKS cluster. One simple example is provided right here, in the sibling folder `eks-cluster`. Another way is making use of ["terraform-aws-modules/eks/aws"](https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/latest). Take note of the cluster's API endpoint URL as well as the cluster's API CA certificate. These will be needed later to configure the Kuberentes provider.

2. Apply this module

    Make sure the same AWS credentials used for the above EKS cluster are avialable in the environment. Provide values for input variables as needed for your use case. For Terraform Cloud, reasonable defaults are baked into the module and all that's required is the name of the TFC Organization that will be used a the "admin group". Identities for all workloads in this org will be granted `cluster-admin` permission on the EKS cluster, via the group name extracted from the configured JWT claim (see input variables). To that end, this module creates a ClusterRoleBinding resource to bind the `cluster-admin` role with the user's group.

You are now ready to access your EKS cluster with indentity tokens provided by Terraform Cloud or your IDp of choice. The Kubernetes provider now only needs to be configured for [host endpoint](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs#host) and [cluster CA](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs#cluster_ca_certificate). In case you are running in Terraform Cloud, the token will automatically be injected into every run's environment (feature not rolled out yet). With other identity providers, you have to collect the token and supply it to the provider using the `KUBE_TOKEN` or the `token` provider attribute.