package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
)

func resourceKubernetesJob() *schema.Resource {
	s := &schema.Resource{
		Create: resourceKubernetesJobCreate,
		Read:   resourceKubernetesJobRead,
		Update: resourceKubernetesJobUpdate,
		Delete: resourceKubernetesJobDelete,
		Exists: resourceKubernetesJobExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
		},
	}

	return s
}

func resourceKubernetesJobCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandJobSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	job := batchv1.Job{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new Job: %#v", job)

	out, err := conn.BatchV1().Jobs(metadata.Namespace).Create(&job)
	if err != nil {
		return fmt.Errorf("Failed to create Job! API error: %s", err)
	}
	log.Printf("[INFO] Submitted new job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesJobRead(d, meta)
}

func resourceKubernetesJobUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating job %s: %#v", d.Id(), ops)

	out, err := conn.BatchV1().Jobs(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update Job! API error: %s", err)
	}
	log.Printf("[INFO] Submitted updated job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	return resourceKubernetesJobRead(d, meta)
}

func resourceKubernetesJobRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading job %s", name)
	job, err := conn.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return fmt.Errorf("Failed to read Job! API error: %s", err)
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
		return err
	}

	jobSpec, err := flattenJobSpec(job.Spec, d)
	if err != nil {
		return err
	}

	err = d.Set("spec", jobSpec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesJobDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting job: %#v", name)
	err = conn.BatchV1().Jobs(namespace).Delete(name, nil)
	if err != nil {
		return fmt.Errorf("Failed to delete Job! API error: %s", err)
	}

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{})
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
		return err
	}

	log.Printf("[INFO] Job %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesJobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking job %s", name)
	_, err = conn.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
