package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesConfigMap() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesConfigMapRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("config_map", false),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the config map data.",
				Computed:    true,
			},
			"binary_data": {
				Type:        schema.TypeMap,
				Description: "A map of the config map binary data.",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesConfigMapRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesConfigMapRead(d, meta)
}
