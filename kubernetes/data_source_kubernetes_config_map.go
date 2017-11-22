package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceKubernetesConfigMap() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesConfigMapRead,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("config map", false),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the configuration data.",
				Computed:    true,
			},
		},
	}
}

func dataSourceKubernetesConfigMapRead(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("metadata.0.name").(string)
	namespace := d.Get("metadata.0.namespace").(string)
	d.SetId(fmt.Sprintf("%s/%s", namespace, name))
	return resourceKubernetesConfigMapRead(d, meta)
}
