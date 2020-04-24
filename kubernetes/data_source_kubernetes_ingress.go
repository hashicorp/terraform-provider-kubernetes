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
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	return resourceKubernetesIngressRead(d, meta)
}
