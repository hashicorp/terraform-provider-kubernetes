// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
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
			"metadata": metadataSchema("EndpointSlice", true),
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
	log.Printf("[INFO] Reading namespace %s", name)
	namespace, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received namespace: %#v", namespace)
	err = d.Set("metadata", flattenMetadata(namespace.ObjectMeta, d, meta))
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

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating namespace: %s", ops)
	out, err := conn.CoreV1().Namespaces().Patch(ctx, d.Id(), pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted updated namespace: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesNamespaceRead(ctx, d, meta)
}

func resourceKubernetesEndpointSliceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Deleting namespace: %#v", name)
	err = conn.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	stateConf := &resource.StateChangeConf{
		Target:  []string{},
		Pending: []string{"Terminating"},
		Timeout: d.Timeout(schema.TimeoutDelete),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
					return nil, "", nil
				}
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Namespace %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Namespace %s deleted", name)

	d.SetId("")
	return nil
}

// func resourceKubernetesNamespaceExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
// 	conn, err := meta.(KubeClientsets).MainClientset()
// 	if err != nil {
// 		return false, err
// 	}

// 	name := d.Id()
// 	log.Printf("[INFO] Checking namespace %s", name)
// 	_, err = conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
// 	if err != nil {
// 		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
// 			return false, nil
// 		}
// 		log.Printf("[DEBUG] Received error: %#v", err)
// 	}
// 	log.Printf("[INFO] Namespace %s exists", name)
// 	return true, err
// }
