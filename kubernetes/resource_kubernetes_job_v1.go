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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func resourceKubernetesJobV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesJobV1Create,
		ReadContext:   resourceKubernetesJobV1Read,
		UpdateContext: resourceKubernetesJobV1Update,
		DeleteContext: resourceKubernetesJobV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceKubernetesJobV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceKubernetesJobUpgradeV0,
			},
		},
		SchemaVersion: 1,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},
		Schema: resourceKubernetesJobV1Schema(),
	}
}

func resourceKubernetesJobV1Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": jobMetadataSchema(),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec of the job owned by the cluster",
			Required:    true,
			MaxItems:    1,
			ForceNew:    false,
			Elem: &schema.Resource{
				Schema: jobSpecFields(false),
			},
		},
		"wait_for_completion": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
	}
}

func resourceKubernetesJobV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandJobV1Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	job := batchv1.Job{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new Job: %#v", job)

	out, err := conn.BatchV1().Jobs(metadata.Namespace).Create(ctx, &job, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create Job! API error: %s", err)
	}
	log.Printf("[INFO] Submitted new job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if d.Get("wait_for_completion").(bool) {
		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
			retryUntilJobV1IsFinished(ctx, conn, namespace, name))
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Diagnostics{}
	}

	return resourceKubernetesJobV1Read(ctx, d, meta)
}

func resourceKubernetesJobV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesJobV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading job %s", name)
	job, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.Errorf("Failed to read Job! API error: %s", err)
	}
	log.Printf("[INFO] Received job: %#v", job)

	// Remove server-generated labels unless using manual selector
	if _, ok := d.GetOk("spec.0.manual_selector"); !ok {
		removeGeneratedLabels(job.ObjectMeta.Labels)
		removeGeneratedLabels(job.Spec.Selector.MatchLabels)
	}

	err = d.Set("metadata", flattenMetadata(job.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	jobSpec, err := flattenJobV1Spec(job.Spec, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", jobSpec)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}

func resourceKubernetesJobV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		specOps := patchJobV1Spec("/spec", "spec.0.", d)
		ops = append(ops, specOps...)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating job %s: %#v", d.Id(), ops)

	out, err := conn.BatchV1().Jobs(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update Job! API error: %s", err)
	}
	log.Printf("[INFO] Submitted updated job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	if d.Get("wait_for_completion").(bool) {
		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate),
			retryUntilJobV1IsFinished(ctx, conn, namespace, name))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceKubernetesJobV1Read(ctx, d, meta)
}

func resourceKubernetesJobV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting job: %#v", name)
	err = conn.BatchV1().Jobs(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.Errorf("Failed to delete Job! API error: %s", err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Job %s still exists", name)
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Job %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesJobV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking job %s", name)
	_, err = conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

// retryUntilJobV1IsFinished checks if a given job has finished its execution in either a Complete or Failed state
func retryUntilJobV1IsFinished(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return func() *retry.RetryError {
		job, err := conn.BatchV1().Jobs(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		for _, c := range job.Status.Conditions {
			if c.Status == corev1.ConditionTrue {
				log.Printf("[DEBUG] Current condition of job: %s/%s: %s\n", ns, name, c.Type)
				switch c.Type {
				case batchv1.JobComplete:
					return nil
				case batchv1.JobFailed:
					return retry.NonRetryableError(fmt.Errorf("job: %s/%s is in failed state", ns, name))
				}
			}
		}

		return retry.RetryableError(fmt.Errorf("job: %s/%s is not in complete state", ns, name))
	}
}
