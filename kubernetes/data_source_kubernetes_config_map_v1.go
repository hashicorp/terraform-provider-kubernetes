// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesConfigMapV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesConfigMapV1Read,

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
			"immutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Immutable, if set to true, ensures that data stored in the ConfigMap cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil.",
			},
		},
	}
}

func dataSourceKubernetesConfigMapV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := metav1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	log.Printf("[INFO] Reading config map %s", metadata.Name)
	cfgMap, err := conn.CoreV1().ConfigMaps(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received config map: %#v", cfgMap)

	err = d.Set("metadata", flattenMetadataFields(cfgMap.ObjectMeta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("binary_data", flattenByteMapToBase64Map(cfgMap.BinaryData))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("data", cfgMap.Data)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("immutable", cfgMap.Immutable)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
