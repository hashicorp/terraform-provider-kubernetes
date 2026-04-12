// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesReferenceGrantV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesReferenceGrantV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("reference_grant_v1", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired state of ReferenceGrant.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from": {
							Type:        schema.TypeList,
							Description: "From describes the trusted namespaces and kinds.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"group":     {Type: schema.TypeString, Computed: true},
									"kind":      {Type: schema.TypeString, Computed: true},
									"namespace": {Type: schema.TypeString, Computed: true},
								},
							},
						},
						"to": {
							Type:        schema.TypeList,
							Description: "To describes the resources that may be referenced.",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"group": {Type: schema.TypeString, Computed: true},
									"kind":  {Type: schema.TypeString, Computed: true},
									"name":  {Type: schema.TypeString, Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesReferenceGrantV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).GatewayClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.Name
	namespace := metadata.Namespace

	log.Printf("[INFO] Reading ReferenceGrant %s", name)
	obj, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[DEBUG] ReferenceGrant %s not found, removing from state", name)
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.Errorf("Failed to read ReferenceGrant '%s' because: %s", name, err)
	}
	log.Printf("[INFO] Received ReferenceGrant: %#v", obj)

	err = d.Set("metadata", flattenMetadata(obj.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSpec := flattenReferenceGrantSpec(obj.Spec)
	log.Printf("[DEBUG] Flattened ReferenceGrant spec: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(obj.ObjectMeta))
	return diag.Diagnostics{}
}
