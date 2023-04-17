// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	//"fmt"
	"log"
	"regexp"

	// "time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	//"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	nodev1 "k8s.io/api/node/v1"

	// api "k8s.io/api/core/v1"
	// "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesRuntimeClassV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesRuntimeClassV1Create,
		ReadContext:   resourceKubernetesRuntimeClassV1Read,
		UpdateContext: resourceKubernetesRuntimeClassV1Update,
		DeleteContext: resourceKubernetesRuntimeClassV1Delete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("runtimeclass", true),
			// "overhead": {
			// 	Type:        schema.TypeMap,
			// 	Description: "Represents the esource overhead associated with running a pod for a given RuntimeClass",
			// },
			// "scheduling": {
			// 	Type:        schema.TypeMap,
			// 	Description: "Holds the scheduling constraints to ensure that pods running with this RuntimeClass are scheduled to nodes that support it",
			// },
			"handler": {
				Type:         schema.TypeString,
				Description:  "Specifies the underlying runtime and configuration that the CRI implementation will use to handle pods of this class",
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?`), ""),
				ForceNew:     true,
			},
		},
	}

}

func resourceKubernetesRuntimeClassV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	//converting metadata for resource -> HCL for TF
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	runtimeClass := nodev1.RuntimeClass{
		ObjectMeta: metadata,
		Handler:    d.Get("handler").(string), //returns the string from handler
	}

	out, err := conn.NodeV1().RuntimeClasses().Create(ctx, &runtimeClass, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] New runtime class created: %#v", out)
	d.SetId(out.Name) //id of resource used in the state file, not a namespace refers to its name of the resource

	return resourceKubernetesRuntimeClassV1Read(ctx, d, meta) //create & read goes hand in hand basically
}

func resourceKubernetesRuntimeClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesRuntimeClassV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading Run Time Class %s", name)
	rc, err := conn.NodeV1().RuntimeClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received Run Time Class: %#v", rc)
	err = d.Set("metadata", flattenMetadata(rc.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("handler", rc.Handler)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesRuntimeClassV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	patch := patchMetadata("metadata.0.", "/metadata/", d)

	data, err := patch.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating run time class %s: %#v", d.Id(), patch)

	out, err := conn.NodeV1().RuntimeClasses().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update run time class! API error: %s", err)
	}

	log.Printf("[INFO] Submitted updated run time class: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesRuntimeClassV1Read(ctx, d, meta)
}

func resourceKubernetesRuntimeClassV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] RunTimeClass: %#v", name)
	err = conn.NodeV1().RuntimeClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}
	log.Printf("[INFO] RunTimeClass %s deleted", name)

	return nil
}

// trying to get resource, if we get an error then we know it doesnt exists
func resourceKubernetesRuntimeClassV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking Run Time Class %s", name)
	_, err = conn.NodeV1().RuntimeClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}

	return true, err
}
