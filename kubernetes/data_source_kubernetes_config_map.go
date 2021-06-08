package kubernetes

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesConfigMap() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesConfigMapRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("config_map", false),
			"data": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "A map of the config map data.",
				Computed:    true,
			},
			"binary_data": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "A map of the config map binary data.",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesConfigMapRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesConfigMapRead(ctx, d, meta)
}
