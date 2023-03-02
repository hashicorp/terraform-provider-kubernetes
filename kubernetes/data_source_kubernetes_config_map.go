// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

func dataSourceKubernetesConfigMapRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesConfigMapRead(ctx, d, meta)
}
