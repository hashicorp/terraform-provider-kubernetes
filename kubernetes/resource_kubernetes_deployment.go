package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesDeployment() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesDeploymentCreate,
		Read:   resourceKubernetesDeploymentRead,
		Exists: resourceKubernetesDeploymentExists,
		Update: resourceKubernetesDeploymentUpdate,
		Delete: resourceKubernetesDeploymentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("deployment", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the desired behavior of the deployment. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min_ready_seconds": {
							Type:        schema.TypeInt,
							Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
							Optional:    true,
							Default:     0,
						},
						"replicas": {
							Type:        schema.TypeInt,
							Description: "The number of desired replicas. Defaults to 1. More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller",
							Optional:    true,
							Default:     1,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Replicas count. If Selector is empty, it is defaulted to the labels present on the Pod template. Label keys and values that must match in order to be controlled by this deployment, if empty defaulted to labels on Pod template. More info: http://kubernetes.io/docs/user-guide/labels#label-selectors",
							Required:    true,
						},
						//todo: strategy not working yet
						"strategy": {
							Type:        schema.TypeMap,
							Optional:    true,
							Computed:    true,
							Description: "Update strategy. One of RollingUpdate, Destroy. Defaults to RollingDate",
						},
						"template": {
							Type:        schema.TypeList,
							Description: "Describes the pod that will be created if insufficient replicas are detected. This takes precedence over a TemplateRef. More info: http://kubernetes.io/docs/user-guide/replication-controller#pod-template",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: podSpecFields(),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesDeploymentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandDeploymentSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	spec.Template.Spec.AutomountServiceAccountToken = ptrToBool(false)

	deployment := v1beta1.Deployment{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new deployment: %#v", deployment)
	out, err := conn.ExtensionsV1beta1().Deployments(metadata.Namespace).Create(&deployment)
	if err != nil {
		return fmt.Errorf("Failed to create deployment: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[DEBUG] Waiting for deployment %s to schedule %d replicas",
		d.Id(), *out.Spec.Replicas)
	// 10 mins should be sufficient for scheduling ~10k replicas
	err = resource.Retry(d.Timeout(schema.TimeoutCreate),
		waitForDeploymentReplicasFunc(conn, out.GetNamespace(), out.GetName()))
	if err != nil {
		return err
	}
	// We could wait for all pods to actually reach Ready state
	// but that means checking each pod status separately (which can be expensive at scale)
	// as there's no aggregate data available from the API

	log.Printf("[INFO] Submitted new deployment: %#v", out)

	return resourceKubernetesDeploymentRead(d, meta)
}

func resourceKubernetesDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading deployment %s", name)
	deployment, err := conn.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received deployment: %#v", deployment)

	err = d.Set("metadata", flattenMetadata(deployment.ObjectMeta))
	if err != nil {
		return err
	}

	spec, err := flattenDeploymentSpec(deployment.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesDeploymentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandDeploymentSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return err
		}

		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating deployment %q: %v", name, string(data))
	out, err := conn.ExtensionsV1beta1().Deployments(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update deployment: %s", err)
	}
	log.Printf("[INFO] Submitted updated deployment: %#v", out)

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate),
		waitForDeploymentReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	return resourceKubernetesDeploymentRead(d, meta)
}

func resourceKubernetesDeploymentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting deployment: %#v", name)

	// Drain all replicas before deleting
	var ops PatchOperations
	ops = append(ops, &ReplaceOperation{
		Path:  "/spec/replicas",
		Value: 0,
	})
	data, err := ops.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = conn.ExtensionsV1beta1().Deployments(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}

	// Wait until all replicas are gone
	err = resource.Retry(d.Timeout(schema.TimeoutDelete),
		waitForDeploymentReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	err = conn.ExtensionsV1beta1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Replication controller %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesDeploymentExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking deployment %s", name)
	_, err := conn.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func waitForDeploymentReplicasFunc(conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		deployment, err := conn.ExtensionsV1beta1().Deployments(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		desiredReplicas := *deployment.Spec.Replicas
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			deployment.GetName(), deployment.Status.Replicas, desiredReplicas)

		if deployment.Status.Replicas == desiredReplicas {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, deployment.GetName(), deployment.Status.Replicas))
	}
}
