package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/apps/v1beta1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesStatefulSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesStatefulSetCreate,
		Read:   resourceKubernetesStatefulSetRead,
		Update: resourceKubernetesStatefulSetUpdate,
		Delete: resourceKubernetesStatefulSetDelete,
		Exists: resourceKubernetesStatefulSetExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("statefulset", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the desired behavior of the StatefulSet. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"replicas": {
							Type:        schema.TypeInt,
							Description: "The number of desired replicas. Defaults to 1. More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller",
							Optional:    true,
							Default:     1,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Replicas count. More info: http://kubernetes.io/docs/user-guide/labels#label-selectors",
							Required:    true,
						},
						"service_name": {
							Type:        schema.TypeString,
							Description: "The name of the service that governs this StatefulSet. This service must exist before the StatefulSet, and is responsible for the network identity of the set. Pods get DNS/hostnames that follow the pattern: pod-specific-string.serviceName.default.svc.cluster.local where \"pod-specific-string\" is managed by the StatefulSet controller.",
							Required:    true,
						},
						"template": {
							Type:        schema.TypeList,
							Description: "Describes the pod that will be created if insufficient replicas are detected. Each pod stamped out by the StatefulSet will fulfill this Template, but have a unique identity from the rest of the StatefulSet.",
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

func resourceKubernetesStatefulSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandStatefulSetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	//use name as label and selector if not set
	if metadata.Name == "" {
		metadata.Name = d.Get("name").(string)
	}
	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}
	if len(spec.Selector.MatchLabels) == 0 {
		spec.Selector.MatchLabels = map[string]string{
			"app": d.Get("name").(string),
		}
		spec.Template.ObjectMeta.Labels = spec.Selector.MatchLabels
	}

	statefulSet := v1beta1.StatefulSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new Stateful Set: %#v", statefulSet)
	out, err := conn.AppsV1beta1().StatefulSets(metadata.Namespace).Create(&statefulSet)
	if err != nil {
		return fmt.Errorf("Failed to create Stateful Set: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[DEBUG] Waiting for Stateful Set %s to schedule %d replicas",
		d.Id(), *out.Spec.Replicas)
	// 10 mins should be sufficient for scheduling ~10k replicas
	err = resource.Retry(d.Timeout(schema.TimeoutCreate),
		waitForStatefulSetReplicasFunc(conn, out.GetNamespace(), out.GetName()))
	if err != nil {
		return err
	}
	// We could wait for all pods to actually reach Ready state
	// but that means checking each pod status separately (which can be expensive at scale)
	// as there's no aggregate data available from the API

	log.Printf("[INFO] Submitted new statefulSet: %#v", out)

	return resourceKubernetesStatefulSetRead(d, meta)
}

func resourceKubernetesStatefulSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading statefulSet %s", name)
	statefulSet, err := conn.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received statefulSet: %#v", statefulSet)

	err = d.Set("metadata", flattenMetadata(statefulSet.ObjectMeta))
	if err != nil {
		return err
	}

	spec, err := flattenStatefulSetSpec(statefulSet.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesStatefulSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandStatefulSetSpec(d.Get("spec").([]interface{}))
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
	log.Printf("[INFO] Updating statefulSet %q: %v", name, string(data))
	out, err := conn.AppsV1beta1().StatefulSets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update statefulSet: %s", err)
	}
	log.Printf("[INFO] Submitted updated statefulSet: %#v", out)

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate),
		waitForStatefulSetReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	return resourceKubernetesStatefulSetRead(d, meta)
}

func resourceKubernetesStatefulSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting statefulSet: %#v", name)

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
	_, err = conn.AppsV1beta1().StatefulSets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}

	// Wait until all replicas are gone
	err = resource.Retry(d.Timeout(schema.TimeoutDelete),
		waitForStatefulSetReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	err = conn.AppsV1beta1().StatefulSets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] StatefulSet %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesStatefulSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking statefulSet %s", name)
	_, err := conn.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func waitForStatefulSetReplicasFunc(conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		statefulSet, err := conn.AppsV1beta1().StatefulSets(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		desiredReplicas := *statefulSet.Spec.Replicas
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			statefulSet.GetName(), statefulSet.Status.Replicas, desiredReplicas)

		if statefulSet.Status.Replicas == desiredReplicas {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, statefulSet.GetName(), statefulSet.Status.Replicas))
	}
}
