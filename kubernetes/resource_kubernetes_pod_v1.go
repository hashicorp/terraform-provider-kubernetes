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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesPodV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesPodV1Create,
		ReadContext:   resourceKubernetesPodV1Read,
		UpdateContext: resourceKubernetesPodV1Update,
		DeleteContext: resourceKubernetesPodV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceKubernetesPodV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceKubernetesPodUpgradeV0,
			},
		},
		SchemaVersion: 1,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: resourceKubernetesPodSchemaV1(),
	}
}

func resourceKubernetesPodSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("pod", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Specification of the desired behavior of the pod.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(false, false),
			},
		},
		"target_state": {
			Type:        schema.TypeList,
			Description: fmt.Sprintf("A list of the pod phases that indicate whether it was successfully created. Options: %q, %q, %q, %q, %q. Default: %q. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase", corev1.PodPending, corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed, corev1.PodUnknown, corev1.PodRunning),
			Optional:    true,
			MinItems:    1,
			Elem: &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					string(corev1.PodPending),
					string(corev1.PodRunning),
					string(corev1.PodSucceeded),
					string(corev1.PodFailed),
					string(corev1.PodUnknown),
				}, false),
			},
		},
	}
}

func resourceKubernetesPodV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	pod := corev1.Pod{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new pod: %#v", pod)
	out, err := conn.CoreV1().Pods(metadata.Namespace).Create(ctx, &pod, metav1.CreateOptions{})

	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new pod: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	stateConf := &retry.StateChangeConf{
		Target:  expandPodTargetState(d.Get("target_state").([]interface{})),
		Pending: []string{string(corev1.PodPending)},
		Timeout: d.Timeout(schema.TimeoutCreate),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().Pods(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Pods %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		lastWarnings, wErr := getLastWarningsForObject(ctx, conn, out.ObjectMeta, "Pod", 3)
		if wErr != nil {
			return diag.FromErr(wErr)
		}
		return diag.Errorf("%s%s", err, stringifyEvents(lastWarnings))
	}
	log.Printf("[INFO] Pod %s created", out.Name)

	return resourceKubernetesPodV1Read(ctx, d, meta)
}

func resourceKubernetesPodV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		specOps, err := patchPodSpec("/spec", "spec.0.", d)
		if err != nil {
			return diag.FromErr(err)
		}
		ops = append(ops, specOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating pod %s: %s", d.Id(), ops)

	out, err := conn.CoreV1().Pods(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted updated pod: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	return resourceKubernetesPodV1Read(ctx, d, meta)
}

func resourceKubernetesPodV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesPodV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading pod %s", name)
	pod, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received pod: %#v", pod)

	err = d.Set("metadata", flattenMetadata(pod.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	podSpec, err := flattenPodSpec(pod.Spec)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", podSpec)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil

}

func resourceKubernetesPodV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting pod: %#v", name)
	err = conn.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		out, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		log.Printf("[DEBUG] Current state of pod: %#v", out.Status.Phase)
		e := fmt.Errorf("Pod %s still exists (%s)", name, out.Status.Phase)
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Pod %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPodV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking pod %s", name)
	_, err = conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
