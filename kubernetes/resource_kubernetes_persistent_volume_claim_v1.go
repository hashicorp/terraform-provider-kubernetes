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
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesPersistentVolumeClaimV1() *schema.Resource {
	fields := persistentVolumeClaimFields()
	// The 'wait_until_bound' control attribute only makes sense in stand-alone PVCs,
	// so adding it on top of the standard PVC fields which are re-usable for other resources.
	fields["wait_until_bound"] = &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Whether to wait for the claim to reach `Bound` state (to find volume in which to claim the space)",
		Optional:    true,
		Default:     true,
	}
	return &schema.Resource{
		CreateContext: resourceKubernetesPersistentVolumeClaimV1Create,
		ReadContext:   resourceKubernetesPersistentVolumeClaimV1Read,
		UpdateContext: resourceKubernetesPersistentVolumeClaimV1Update,
		DeleteContext: resourceKubernetesPersistentVolumeClaimV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("wait_until_bound", true)
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: fields,

		// All fields of Spec are immutable after creation, except for resources.requests.storage.
		// Storage can only be increased in place. A new object will be created when the storage is decreased.
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			// Skip custom logic for resource creation.
			if diff.Id() == "" {
				return nil
			}
			key := "spec.0.resources.0.requests"
			subKeyStorage := "spec.0.resources.0.requests.storage"
			subKeyLimits := "spec.0.resources.0.limits"
			if diff.HasChange(subKeyLimits) {
				err := diff.ForceNew(subKeyLimits)
				if err != nil {
					return err
				}
				return nil
			}
			if diff.HasChange(key) {
				old, new := diff.GetChange(subKeyStorage)
				oldStorageQuantity, err := k8sresource.ParseQuantity(old.(string))
				if err != nil {
					return err
				}
				newStorageQuantity, err := k8sresource.ParseQuantity(new.(string))
				if err != nil {
					return err
				}
				if newStorageQuantity.Cmp(oldStorageQuantity) == -1 {
					log.Printf("[DEBUG] CustomizeDiff spec.resources.requests.storage: field can not be less than previous value")
					log.Printf("[DEBUG] CustomizeDiff creating new PVC with size: %v", new)
					err := diff.ForceNew(key)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func resourceKubernetesPersistentVolumeClaimV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	claim, err := expandPersistentVolumeClaim(map[string]interface{}{
		"metadata": d.Get("metadata"),
		"spec":     d.Get("spec"),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Creating new persistent volume claim: %#v", claim)
	out, err := conn.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(ctx, claim, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new persistent volume claim: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	name := out.ObjectMeta.Name

	if d.Get("wait_until_bound").(bool) {
		stateConf := &retry.StateChangeConf{
			Target:  []string{"Bound"},
			Pending: []string{"Pending"},
			Timeout: d.Timeout(schema.TimeoutCreate),
			Refresh: func() (interface{}, string, error) {
				out, err := conn.CoreV1().PersistentVolumeClaims(claim.Namespace).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					log.Printf("[ERROR] Received error: %#v", err)
					return out, "", err
				}

				statusPhase := fmt.Sprintf("%v", out.Status.Phase)
				log.Printf("[DEBUG] Persistent volume claim %s status received: %#v", out.Name, statusPhase)
				return out, statusPhase, nil
			},
		}
		_, err = stateConf.WaitForStateContext(ctx)
		if err != nil {
			var lastWarnings []api.Event
			var wErr error

			lastWarnings, wErr = getLastWarningsForObject(ctx, conn, out.ObjectMeta, "PersistentVolumeClaim", 3)
			if wErr != nil {
				return diag.FromErr(wErr)
			}

			if len(lastWarnings) == 0 {
				lastWarnings, wErr = getLastWarningsForObject(ctx, conn, metav1.ObjectMeta{
					Name: out.Spec.VolumeName,
				}, "PersistentVolume", 3)
				if wErr != nil {
					return diag.FromErr(wErr)
				}
			}

			return diag.Errorf("%s%s", err, stringifyEvents(lastWarnings))
		}
	}
	log.Printf("[INFO] Persistent volume claim %s created", out.Name)

	return resourceKubernetesPersistentVolumeClaimV1Read(ctx, d, meta)
}

func resourceKubernetesPersistentVolumeClaimV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesPersistentVolumeClaimV1Exists(ctx, d, meta)
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

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading persistent volume claim %s", name)
	claim, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received persistent volume claim: %#v", claim)
	err = d.Set("metadata", flattenMetadata(claim.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("spec", flattenPersistentVolumeClaimSpec(claim.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesPersistentVolumeClaimV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	// spec.resources.requests is the only editable field in Spec.
	if d.HasChange("spec.0.resources.0.requests") {
		r := d.Get("spec.0.resources.0.requests").(map[string]interface{})
		requests, err := expandMapToResourceList(r)
		if err != nil {
			return diag.FromErr(err)
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec/resources/requests",
			Value: requests,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating persistent volume claim: %s", ops)
	out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted updated persistent volume claim: %#v", out)

	return resourceKubernetesPersistentVolumeClaimV1Read(ctx, d, meta)
}

func resourceKubernetesPersistentVolumeClaimV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting persistent volume claim: %#v", name)
	err = conn.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		log.Printf("[DEBUG] Current state of persistent volume claim finalizers: %#v", out.Finalizers)
		e := fmt.Errorf("Persistent volume claim %s still exists with finalizers: %v", name, out.Finalizers)
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Persistent volume claim %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPersistentVolumeClaimV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking persistent volume claim %s", name)
	_, err = conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
