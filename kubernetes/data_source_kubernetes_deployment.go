package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesDeployment() *schema.Resource {
	dsSchema := datasourceSchemaFromResourceSchema(resourceKubernetesDeployment().Schema)

	addRequiredFieldsToSchema(dsSchema, "metadata")
	addRequiredFieldsToSchema(dsSchema["metadata"].Elem.(*schema.Resource).Schema, "name")
	addRequiredFieldsToSchema(dsSchema["metadata"].Elem.(*schema.Resource).Schema, "namespace")

	return &schema.Resource{
		Read: dataSourceKubernetesDeploymentRead,

		Schema: dsSchema,
	}
}

func dataSourceKubernetesDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesDeploymentRead(d, meta)
}
