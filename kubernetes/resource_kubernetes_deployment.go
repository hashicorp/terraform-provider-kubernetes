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
	kubernetes "k8s.io/client-go/kubernetes"
	api "k8s.io/client-go/pkg/api/v1"
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
							Type:         schema.TypeInt,
							Description:  "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
							Optional:     true,
							Default:      0,
							ValidateFunc: validatePositiveInteger,
						},
						"paused": {
							Type:        schema.TypeBool,
							Description: "Whether the deployment is paused",
							Optional:    true,
							Default:     false,
						},
						"progress_deadline_seconds": {
							Type:         schema.TypeInt,
							Description:  "The maximum time in seconds for a deployment to make progress before it is considered to be failed.",
							Optional:     true,
							Default:      600,
							ValidateFunc: validatePositiveInteger,
						},
						"replicas": {
							Type:         schema.TypeInt,
							Description:  "The number of desired replicas. Defaults to 1.",
							Optional:     true,
							Default:      1,
							ValidateFunc: validatePositiveInteger,
						},
						"revision_history_limit": {
							Type:         schema.TypeInt,
							Description:  "The number of old ReplicaSets to retain to allow rollback.",
							Optional:     true,
							Default:      10,
							ValidateFunc: validatePositiveInteger,
						},
						"selector": labelSelectorSchema("pods"),
						"strategy": {
							Type:        schema.TypeList,
							Description: "The deployment strategy to use to replace existing pods with new ones.",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"rolling_update": {
										Type:        schema.TypeList,
										Description: "Rolling update config params. Present only if strategy type = 'RollingUpdate'.",
										Optional:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_surge": {
													Description: "The maximum number of pods that can be scheduled above the desired number of pods. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if max_unavailable is 0. Absolute number is calculated from percentage by rounding up. Defaults to 25%",
													Default:     "25%",
												},
												"max_unavailable": {
													Description: "The maximum number of pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). Absolute number is calculated from percentage by rounding down. This can not be 0 if max_surge is 0. Defaults to 25%.",
													Default:     "25%",
												},
											},
										},
									},
									"type": {
										Type:         schema.TypeString,
										Description:  `Type of deployment. Can be "Recreate" or "RollingUpdate".`,
										Optional:     true,
										ValidateFunc: validateDeploymentStrategyType,
										Default:      "RollingUpdate",
									},
								},
							},
						},
						"template": {
							Type:        schema.TypeList,
							Description: "Template describes the pods that will be created.",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: podSpecFields(true),
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
	spec, err := expandReplicationControllerSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	spec.Template.Spec.AutomountServiceAccountToken = ptrToBool(false)

	rc := api.ReplicationController{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new replication controller: %#v", rc)
	out, err := conn.CoreV1().ReplicationControllers(metadata.Namespace).Create(&rc)
	if err != nil {
		return fmt.Errorf("Failed to create replication controller: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[DEBUG] Waiting for replication controller %s to schedule %d replicas",
		d.Id(), *out.Spec.Replicas)
	// 10 mins should be sufficient for scheduling ~10k replicas
	err = resource.Retry(d.Timeout(schema.TimeoutCreate),
		waitForDesiredReplicasFunc(conn, out.GetNamespace(), out.GetName()))
	if err != nil {
		return err
	}
	// We could wait for all pods to actually reach Ready state
	// but that means checking each pod status separately (which can be expensive at scale)
	// as there's no aggregate data available from the API

	log.Printf("[INFO] Submitted new replication controller: %#v", out)

	return resourceKubernetesReplicationControllerRead(d, meta)
}

func resourceKubernetesReplicationControllerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading replication controller %s", name)
	rc, err := conn.CoreV1().ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received replication controller: %#v", rc)

	err = d.Set("metadata", flattenMetadata(rc.ObjectMeta))
	if err != nil {
		return err
	}

	spec, err := flattenReplicationControllerSpec(rc.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesReplicationControllerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandReplicationControllerSpec(d.Get("spec").([]interface{}))
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
	log.Printf("[INFO] Updating replication controller %q: %v", name, string(data))
	out, err := conn.CoreV1().ReplicationControllers(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update replication controller: %s", err)
	}
	log.Printf("[INFO] Submitted updated replication controller: %#v", out)

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate),
		waitForDesiredReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	return resourceKubernetesReplicationControllerRead(d, meta)
}

func resourceKubernetesReplicationControllerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting replication controller: %#v", name)

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
	_, err = conn.CoreV1().ReplicationControllers(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}

	// Wait until all replicas are gone
	err = resource.Retry(d.Timeout(schema.TimeoutDelete),
		waitForDesiredReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	err = conn.CoreV1().ReplicationControllers(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Replication controller %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesReplicationControllerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking replication controller %s", name)
	_, err = conn.CoreV1().ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func waitForDesiredReplicasFunc(conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		rc, err := conn.CoreV1().ReplicationControllers(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		desiredReplicas := *rc.Spec.Replicas
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			rc.GetName(), rc.Status.FullyLabeledReplicas, desiredReplicas)

		if rc.Status.FullyLabeledReplicas == desiredReplicas {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, rc.GetName(), rc.Status.FullyLabeledReplicas))
	}
}
