// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesNamespaceV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesNamespaceV1Create,
		ReadContext:   resourceKubernetesNamespaceV1Read,
		UpdateContext: resourceKubernetesNamespaceV1Update,
		DeleteContext: resourceKubernetesNamespaceV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("namespace", true),
			"wait_for_default_service_account": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Terraform will wait for the default service account to be created.",
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
	}
}

func resourceKubernetesNamespaceV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	namespace := corev1.Namespace{
		ObjectMeta: metadata,
	}
	log.Printf("[INFO] Creating new namespace: %#v", namespace)
	out, err := conn.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new namespace: %#v", out)
	d.SetId(out.Name)

	if d.Get("wait_for_default_service_account").(bool) {
		log.Printf("[DEBUG] Waiting for default service account to be created")
		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
			_, err := conn.CoreV1().ServiceAccounts(out.Name).Get(ctx, "default", metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					log.Printf("[INFO] Default service account does not exist, will retry: %s", metadata.Namespace)
					return retry.RetryableError(err)
				}

				return retry.NonRetryableError(err)
			}

			log.Printf("[INFO] Default service account exists: %s", metadata.Namespace)
			return nil
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceKubernetesNamespaceV1Read(ctx, d, meta)
}

func resourceKubernetesNamespaceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesNamespaceV1Exists(ctx, d, meta)
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

func resourceKubernetesNamespaceV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	return resourceKubernetesNamespaceV1Read(ctx, d, meta)
}

func resourceKubernetesNamespaceV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	stateConf := &retry.StateChangeConf{
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

func resourceKubernetesNamespaceV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
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
