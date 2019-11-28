package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesNamespace() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesNamespaceRead,

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("namespace", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of the Namespace.",
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"finalizers": {
							Type:        schema.TypeList,
							Description: "Finalizers is an opaque list of values that must be empty to permanently remove object from storage.",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("metadata.0.name").(string)
	d.SetId(name)
	conn := meta.(*KubeClientsets).MainClientset

	log.Printf("[INFO] Reading namespace %s", name)
	namespace, err := conn.CoreV1().Namespaces().Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received namespace: %#v", namespace)
	err = d.Set("metadata", flattenMetadata(namespace.ObjectMeta, d))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenNamespaceSpec(&namespace.Spec))
	if err != nil {
		return err
	}
	return nil
}

func flattenNamespaceSpec(in *v1.NamespaceSpec) []interface{} {
	if in == nil || len(in.Finalizers) == 0 {
		return []interface{}{}
	}
	spec := make(map[string]interface{})
	fin := make([]string, len(in.Finalizers))
	for i, f := range in.Finalizers {
		fin[i] = string(f)
	}
	spec["finalizers"] = fin
	return []interface{}{spec}
}
