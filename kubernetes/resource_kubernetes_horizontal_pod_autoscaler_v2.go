package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesHorizontalPodAutoscalerV2Create(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandHorizontalPodAutoscalerV2Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	hpa := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new horizontal pod autoscaler: %#v", hpa)
	out, err := conn.AutoscalingV2beta2().HorizontalPodAutoscalers(metadata.Namespace).Create(ctx, &hpa, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted new horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerV2Read(d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerV2Read(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Reading horizontal pod autoscaler %s", name)
	hpa, err := conn.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received horizontal pod autoscaler: %#v", hpa)
	err = d.Set("metadata", flattenMetadata(hpa.ObjectMeta, d))
	if err != nil {
		return err
	}

	flattened := flattenHorizontalPodAutoscalerV2Spec(hpa.Spec)
	log.Printf("[DEBUG] Flattened horizontal pod autoscaler spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesHorizontalPodAutoscalerV2Update(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps := patchHorizontalPodAutoscalerV2Spec("spec.0.", "/spec", d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating horizontal pod autoscaler %q: %v", name, string(data))
	out, err := conn.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update horizontal pod autoscaler: %s", err)
	}
	log.Printf("[INFO] Submitted updated horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerV2Read(d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerV2Delete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleting horizontal pod autoscaler: %#v", name)
	err = conn.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Horizontal Pod Autoscaler %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesHorizontalPodAutoscalerV2Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking horizontal pod autoscaler %s", name)
	_, err = conn.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
