package v1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DataSourceKubernetesSecret() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesSecretRead,

		Schema: map[string]*schema.Schema{
			"metadata": providermetav1.NamespacedMetadataSchema("secret", false),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Computed:    true,
				Sensitive:   true,
			},
			"binary_data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data with values encoded in base64 format",
				Optional:    true,
				Sensitive:   true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Computed:    true,
			},
			"immutable": {
				Type:        schema.TypeBool,
				Description: "Ensures that data stored in the Secret cannot be updated (only object metadata can be modified).",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	om := metav1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(providermetav1.BuildId(om))

	return resourceKubernetesSecretRead(ctx, d, meta)
}
