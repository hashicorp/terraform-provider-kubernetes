// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	api "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesEndpointSliceV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesEndpointSliceV1Create,
		ReadContext:   resourceKubernetesEndpointSliceV1Read,
		UpdateContext: resourceKubernetesEndpointSliceV1Update,
		DeleteContext: resourceKubernetesEndpointSliceV1Delete,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("endpoint_slice", true),
			"address_type": {
				Type:         schema.TypeString,
				Description:  "address_type specifies the type of address carried by this EndpointSlice. All addresses in this slice must be the same type. This field is immutable after creation.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"IPv4", "IPv6", "FQDN"}, false),
			},
			"endpoint": {
				Description: "endpoint is a list of unique endpoints in this slice. Each slice may include a maximum of 1000 endpoints.",
				Type:        schema.TypeList,
				MaxItems:    1000,
				Required:    true,
				Elem:        schemaEndpointSliceSubsetEndpoints(),
			},
			"port": {
				Description: "port specifies the list of network ports exposed by each endpoint in this slice. Each port must have a unique name. Each slice may include a maximum of 100 ports.",
				Type:        schema.TypeList,
				MaxItems:    100,
				Required:    true,
				Elem:        schemaEndpointSliceSubsetPorts(),
			},
		},
	}
}

func resourceKubernetesEndpointSliceV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	endpoint_slice := api.EndpointSlice{
		ObjectMeta:  metadata,
		AddressType: api.AddressType(d.Get("address_type").(string)),
		Endpoints:   expandEndpointSliceEndpoints(d.Get("endpoint").([]interface{})),
		Ports:       expandEndpointSlicePorts(d.Get("port").([]interface{})),
	}

	log.Printf("[INFO] Creating new endpoint_slice: %#v", endpoint_slice)
	out, err := conn.DiscoveryV1().EndpointSlices(metadata.Namespace).Create(ctx, &endpoint_slice, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create endpoint_slice because: %s", err)
	}
	log.Printf("[INFO] Submitted new endpoint_slice: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointSliceV1Read(ctx, d, meta)
}

func resourceKubernetesEndpointSliceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading endpoint slice %s", name)
	endpoint, err := conn.DiscoveryV1().EndpointSlices(namespace).Get(ctx, name, metav1.GetOptions{})
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

	flattenedEndpoints := flattenEndpointSliceEndpoints(endpoint.Endpoints)
	log.Printf("[DEBUG] Flattened EndpointSlice Endpoints: %#v", flattenedEndpoints)
	err = d.Set("endpoint", flattenedEndpoints)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedPorts := flattenEndpointSlicePorts(endpoint.Ports)
	log.Printf("[DEBUG] Flattened EndpointSlice Ports: %#v", flattenedPorts)
	err = d.Set("port", flattenedPorts)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesEndpointSliceV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.Errorf("Failed to update endpointSlice because: %s", err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("address_type") {
		address_type := d.Get("address_type").(string)
		ops = append(ops, &ReplaceOperation{
			Path:  "/addressType",
			Value: address_type,
		})
	}
	if d.HasChange("endpoint") {
		endpoints := expandEndpointSliceEndpoints(d.Get("endpoint").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  "/endpoints",
			Value: endpoints,
		})
	}
	if d.HasChange("port") {
		ports := expandEndpointSlicePorts(d.Get("port").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  "/ports",
			Value: ports,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating endpointSlice %q: %v", name, string(data))
	out, err := conn.DiscoveryV1().EndpointSlices(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update endpointSlice: %s", err)
	}
	log.Printf("[INFO] Submitted updated endpointSlice: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointSliceV1Read(ctx, d, meta)
}

func resourceKubernetesEndpointSliceV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
