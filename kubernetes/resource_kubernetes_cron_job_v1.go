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
	"k8s.io/apimachinery/pkg/api/errors"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesCronJobV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesCronJobV1Create,
		ReadContext:   resourceKubernetesCronJobV1Read,
		UpdateContext: resourceKubernetesCronJobV1Update,
		DeleteContext: resourceKubernetesCronJobV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("cronjob", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the cron job owned by the cluster",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: cronJobSpecFieldsV1(),
				},
			},
		},
	}
}

func resourceKubernetesCronJobV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandCronJobSpecV1(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	job := batch.CronJob{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	log.Printf("[INFO] Creating new cron job: %#v", job)

	out, err := conn.BatchV1().CronJobs(metadata.Namespace).Create(ctx, &job, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new cron job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesCronJobV1Read(ctx, d, meta)
}

func resourceKubernetesCronJobV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, _, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandCronJobSpecV1(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	cronjob := &batch.CronJob{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Updating cron job %s: %s", d.Id(), cronjob)

	out, err := conn.BatchV1().CronJobs(namespace).Update(ctx, cronjob, metav1.UpdateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted updated cron job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	return resourceKubernetesCronJobV1Read(ctx, d, meta)
}

func resourceKubernetesCronJobV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesCronJobV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading cron job %s", name)
	job, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received cron job: %#v", job)

	// Remove server-generated labels unless using manual selector
	if _, ok := d.GetOk("spec.0.job_template.spec.0.manual_selector"); !ok {
		removeGeneratedLabels(job.ObjectMeta.Labels)
		if job.Spec.JobTemplate.Spec.Selector != nil {
			removeGeneratedLabels(job.Spec.JobTemplate.Spec.Selector.MatchLabels)
		}
	}

	err = d.Set("metadata", flattenMetadata(job.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	jobSpec, err := flattenCronJobSpecV1(job.Spec, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", jobSpec)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesCronJobV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting cron job: %#v", name)
	err = conn.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Cron Job %s still exists", name)
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Cron Job %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesCronJobV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking cron job %s", name)
	_, err = conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
