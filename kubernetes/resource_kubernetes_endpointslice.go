// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesEndpointSlice() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesEndpointSliceCreate,
		ReadContext:   resourceKubernetesEndpointSliceRead,
		UpdateContext: resourceKubernetesEndpointSliceUpdate,
		DeleteContext: resourceKubernetesEndpointSliceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("endpointSlice", true),
			"address_type": {
				Type:        schema.TypeString,
				Description: "addressType specifies the type of address carried by this EndpointSlice. All addresses in this slice must be the same type.",
				Required:    true,
			},
			"endpoints": {
				Type:        schema.TypeList,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
				Required:    true,
				Elem:        schemaEndpointSliceSubsetEndpoints(),
			},
			"ports": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     schemaEndpointSliceSubsetPorts(),
			},
		},
	}
}

func resourceKubernetesEndpointSliceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	endpoint_slice := api.EndpointSlice{
		ObjectMeta:  metadata,
		AddressType: api.AddressType(d.Get("address_type").(string)),
		Endpoints:   expandEndpointSliceEndpoints(d.Get("endpoints").(*schema.Set)),
		Ports:       expandEndpointSlicePorts(d.Get("ports").(*schema.Set)),
	}

	log.Printf("[INFO] Creating new endpoint_slice: %#v", endpoint_slice)
	out, err := conn.DiscoveryV1().EndpointSlices(metadata.Namespace).Create(ctx, &endpoint_slice, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create endpoint_slice because: %s", err)
	}

	log.Printf("[INFO] Submitted new endpoint_slice: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointSliceRead(ctx, d, meta)
}

func resourceKubernetesEndpointSliceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesNamespaceExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	log.Printf("[INFO] Reading endpoint slice %s", name)
	endpoint, err := conn.DiscoveryV1().EndpointSlices(metadata.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.Errorf("Failed to read endpoint_slice because: %s", err)
	}
	log.Printf("[INFO] Received endpoint slice: %#v", endpoint)

	address_type := d.Get("address_type").(string)
	log.Printf("[DEBUG] Default address type is %q", address_type)
	d.Set("address_type", address_type)

	err = d.Set("metadata", flattenMetadata(endpoint.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenEndpointSliceEndpoints(endpoint.Endpoints)
	log.Printf("[DEBUG] Flattened EndpointSlice Endpoints: %#v", flattened)
	err = d.Set("endpoints", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	flattened = flattenEndpointSlicePorts(endpoint.Ports)
	log.Printf("[DEBUG] Flattened EndpointSlice Ports: %#v", flattened)
	err = d.Set("endpoints", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesEndpointSliceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.Errorf("Failed to update endpointSlice because: %s", err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("endpoints") {
		endpoints := expandEndpointSliceEndpoints(d.Get("endpoints").(*schema.Set))
		ops = append(ops, &ReplaceOperation{
			Path:  "/endpoints",
			Value: endpoints,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating endpointSlice %q: %v", name, string(data))
	out, err := conn.DiscoveryV1().EndpointSlices(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.Errorf("Failed to update endpointSlice: %s", err)
	}
	log.Printf("[INFO] Submitted updated endpointSlice: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesNamespaceRead(ctx, d, meta)
}

func resourceKubernetesEndpointSliceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.Errorf("Failed to delete endpointSlice because: %s", err)
	}
	log.Printf("[INFO] Deleting endpointSlice: %#v", name)
	err = conn.DiscoveryV1().EndpointSlices(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] EndpointSlice %s deleted", name)
	d.SetId("")

	return nil
}
