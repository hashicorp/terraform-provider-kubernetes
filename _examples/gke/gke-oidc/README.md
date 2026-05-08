# Summary

This module configures a GKE cluster to use Terraform Cloud or Terraform Enterprise as an OIDC identity provider.

# Usage

This module requires a GKE cluster that is up-and-running and has identity services enabled.
If you already have a GKE , but are unsure if identity services are enabled, you can enable it as described here: https://cloud.google.com/kubernetes-engine/docs/how-to/oidc#enabling_on_a_new_cluster

If you provisioned your cluster with Terraform, you can add the following block to your `resource "google_container_cluster"` configuration and re-apply:

```
  identity_service_config {
    enabled = true
  }
```

Applying this module will modify an existing custom resource already present in the target cluster, adding the necessary details specific to TFC / TFE. The `authentication.gke.io.v2alpha1.ClientConfig` custom resoruce will only be present in cluster whrere identity service has been enabled (as described above).

To configure the module, the following Terraform variables need to be set. Variables with a default value are optional.

| Variable         | Contents                                                                                     | Default value               |
| ---------------- | -------------------------------------------------------------------------------------------- | --------------------------- |
| cluster_name     | Name of the target GKE cluster to configure                                                  |                             |
| gke_location     | Location (zone or region) of the cluster in GCP                                              |                             |
| odic_issuer_uri  | Base URL of TFC / TFE endpoint (default to public TFC)                                       | https://app.terraform.io    |
| oidc_audience    | Audience value as configured in TFC / TFE environment variable                               | kubernetes                  |
| oidc_user_claim  | Token claim to extract user name from (defaults to 'sub')                                    | sub                         |
| oidc_group_claim | Token claim to extract the group membership from (defaults to 'terraform_organization_name') | terraform_organization_name |
| TFE_CA_cert      | CA Certificate for the HTTPS API endpoint of Terraform Enterprise (contents, not filepath)   |                             |

**BEWARE** _Once this module is successfully applied, the `authentication.gke.io.v2alpha1.ClientConfig` CR named "default" in namespace "kube-public" becomes managed by Terraform, as is usual with imported resources. As a consequence, destroying this module will also remove that resource from the cluster._
