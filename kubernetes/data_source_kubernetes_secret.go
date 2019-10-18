package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesSecret() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesSecretRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("secret", false),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Computed:    true,
				Sensitive:   true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesSecretRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesSecretRead(d, meta)
}
