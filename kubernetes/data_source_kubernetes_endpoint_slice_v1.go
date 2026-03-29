// Copyright (c) IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesEndpointSliceV1() *schema.Resource {
	return &schema.Resource{
		Description: "An EndpointSlice contains references to a set of network endpoints. This data source allows you to pull data about such endpoint slice.",
		ReadContext: dataSourceKubernetesEndpointSliceV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("endpoint_slice", true),
			"address_type": {
				Type:        schema.TypeString,
				Description: "address_type specifies the type of address carried by this EndpointSlice. All addresses in this slice must be the same type. This field is immutable after creation.",
				Computed:    true,
			},
			"endpoint": {
				Type:        schema.TypeList,
				Description: "endpoint is a list of unique endpoints in this slice. Each slice may include a maximum of 1000 endpoints.",
				Computed:    true,
				Elem:        schemaEndpointSliceSubsetEndpoints(),
			},
			"port": {
				Type:        schema.TypeList,
				Description: "port specifies the list of network ports exposed by each endpoint in this slice. Each port must have a unique name. Each slice may include a maximum of 100 ports.",
				Computed:    true,
				Elem:        schemaEndpointSliceSubsetPorts(),
			},
		},
	}
}

func dataSourceKubernetesEndpointSliceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	log.Printf("[INFO] Reading endpoint slice %s", metadata.Name)
	ep, err := conn.DiscoveryV1().EndpointSlices(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.Errorf("Failed to read endpoint slice because: %s", err)
	}
	log.Printf("[INFO] Received endpoint slice: %#v", ep)

	err = d.Set("metadata", flattenMetadataFields(ep.ObjectMeta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("address_type", string(ep.AddressType))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("endpoint", flattenEndpointSliceEndpoints(ep.Endpoints))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("port", flattenEndpointSlicePorts(ep.Ports))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
