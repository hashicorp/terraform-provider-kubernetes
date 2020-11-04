package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func resourceKubernetesJob() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesJobCreate,
		ReadContext:   resourceKubernetesJobRead,
		UpdateContext: resourceKubernetesJobUpdate,
		DeleteContext: resourceKubernetesJobDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": jobMetadataSchema(),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the job owned by the cluster",
				Required:    true,
				MaxItems:    1,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: jobSpecFields(),
				},
			},
			"wait_for_completion": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceKubernetesJobCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandJobSpec(d.Get("spec").([]interface{}))
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
		err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
			retryUntilJobIsFinished(ctx, conn, namespace, name))
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Diagnostics{}
	}

	return resourceKubernetesJobRead(ctx, d, meta)
}

func resourceKubernetesJobUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

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
		err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate),
			retryUntilJobIsFinished(ctx, conn, namespace, name))
		return diag.FromErr(err)
	}
	return resourceKubernetesJobRead(ctx, d, meta)
}

func resourceKubernetesJobRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesJobExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
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
		labels := job.ObjectMeta.Labels

		if _, ok := labels["controller-uid"]; ok {
			delete(labels, "controller-uid")
		}

		if _, ok := labels["job-name"]; ok {
			delete(labels, "job-name")
		}

		labels = job.Spec.Selector.MatchLabels

		if _, ok := labels["controller-uid"]; ok {
			delete(labels, "controller-uid")
		}
	}

	err = d.Set("metadata", flattenMetadata(job.ObjectMeta, d))
	if err != nil {
		return diag.FromErr(err)
	}

	jobSpec, err := flattenJobSpec(job.Spec, d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", jobSpec)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}

func resourceKubernetesJobDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		return diag.Errorf("Failed to delete Job! API error: %s", err)
	}

	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		e := fmt.Errorf("Job %s still exists", name)
		return resource.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Job %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesJobExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
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
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

// retryUntilJobIsFinished checks if a give job finished its execution and either in Complete or Failed state
func retryUntilJobIsFinished(ctx context.Context, conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		job, err := conn.BatchV1().Jobs(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		for _, c := range job.Status.Conditions {
			if c.Status == corev1.ConditionTrue {
				log.Printf("[DEBUG] Current condition of job: %s/%s: %s\n", ns, name, c.Type)
				switch c.Type {
				case batchv1.JobComplete:
					return nil
				case batchv1.JobFailed:
					return resource.NonRetryableError(fmt.Errorf("job: %s/%s is in failed state", ns, name))
				}
			}
		}

		return resource.RetryableError(fmt.Errorf("job: %s/%s is not in complete state", ns, name))
	}
}
