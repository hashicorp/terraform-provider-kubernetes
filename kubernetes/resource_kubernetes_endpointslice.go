// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesNamespaceCreate,
		ReadContext:   resourceKubernetesNamespaceRead,
		UpdateContext: resourceKubernetesNamespaceUpdate,
		DeleteContext: resourceKubernetesNamespaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("EndpointSlice", true),
			"address_type": {
				Type:        schema.TypeString,
				Description: "addressType specifies the type of address carried by this EndpointSlice. All addresses in this slice must be the same type.",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Optional:    true,
						},
					},
				},
			},
			"endpoints": {
				Type:        schema.TypeList,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"addresses": {
							Type:        schema.TypeList,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"conditions": {
							Type:        schema.TypeList,
							Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ready": {
										Type:        schema.TypeBool,
										Description: "Specification of the desired behavior of the job",
										Optional:    true,
									},
									"serving": {
										Type:        schema.TypeBool,
										Description: "Specification of the desired behavior of the job",
										Optional:    true,
									},
									"terminating": {
										Type:        schema.TypeBool,
										Description: "Specification of the desired behavior of the job",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
	}
}

func resourceKubernetesNamespaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	namespace := api.Namespace{
		ObjectMeta: metadata,
	}
	log.Printf("[INFO] Creating new namespace: %#v", namespace)
	out, err := conn.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new namespace: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesNamespaceRead(ctx, d, meta)
}

func resourceKubernetesNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func resourceKubernetesNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func resourceKubernetesNamespaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func resourceKubernetesNamespaceExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()
	log.Printf("[INFO] Checking namespace %s", name)
	_, err = conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	log.Printf("[INFO] Namespace %s exists", name)
	return true, err
}
