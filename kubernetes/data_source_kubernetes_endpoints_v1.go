// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesEndpointsV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesEndpointsV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("endpoints", true),
			"subset": {
				Type:        schema.TypeSet,
				Description: "Set of addresses and ports that comprise a service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors",
				Optional:    true,
				Elem:        schemaEndpointsSubset(),
				Set:         hashEndpointsSubset(),
			},
		},
	}
}

func dataSourceKubernetesEndpointsV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	om := metav1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesEndpointsV1Read(ctx, d, meta)
}
