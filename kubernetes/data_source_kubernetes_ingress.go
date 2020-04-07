package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesIngress() *schema.Resource {
	s := resourceKubernetesIngress().Schema
	s["metadata"] = namespacedMetadataSchema("ingress", false)
	s["spec"].Computed = true
	s["spec"].Required = false
	return &schema.Resource{
		Read:   dataSourceKubernetesIngressRead,
		Schema: s,
	}
}

func dataSourceKubernetesIngressRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesIngressRead(d, meta)
}
