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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func resourceKubernetesReplicationControllerV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesReplicationControllerV1Create,
		ReadContext:   resourceKubernetesReplicationControllerV1Read,
		UpdateContext: resourceKubernetesReplicationControllerV1Update,
		DeleteContext: resourceKubernetesReplicationControllerV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceKubernetesReplicationControllerV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceKubernetesReplicationControllerUpgradeV0,
			},
		},
		SchemaVersion: 1,
		Schema:        resourceKubernetesReplicationControllerV1Schema(),
	}
}

func resourceKubernetesReplicationControllerV1Schema() map[string]*schema.Schema {

	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("replication controller", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the specification of the desired behavior of the replication controller. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
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
						Description: "The number of desired replicas. Defaults to 1. More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#what-is-a-replicationcontroller",
						Optional:    true,
						Default:     1,
					},
					"selector": {
						Type:        schema.TypeMap,
						Description: "A label query over pods that should match the Replicas count. If Selector is empty, it is defaulted to the labels present on the Pod template. Label keys and values that must match in order to be controlled by this replication controller, if empty defaulted to labels on Pod template. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors",
						Required:    true,
					},
					"template": {
						Type:        schema.TypeList,
						Description: "Describes the pod that will be created if insufficient replicas are detected. This takes precedence over a TemplateRef. More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#pod-template",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: replicationControllerTemplateFieldSpec(),
						},
					},
				},
			},
		},
	}
}

func replicationControllerTemplateFieldSpec() map[string]*schema.Schema {
	metadata := namespacedMetadataSchemaIsTemplate("replication controller's template", true, true)
	metadata.Required = true

	templateFields := map[string]*schema.Schema{
		"metadata": metadata,
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec of the pods managed by the replication controller",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(false, true),
			},
		},
	}
	return templateFields
}

func resourceKubernetesReplicationControllerV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	spec, err := expandReplicationControllerSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	rc := api.ReplicationController{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new replication controller: %#v", rc)
	out, err := conn.CoreV1().ReplicationControllers(metadata.Namespace).Create(ctx, &rc, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create replication controller: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[DEBUG] Waiting for replication controller %s to schedule %d replicas",
		d.Id(), *out.Spec.Replicas)
	// 10 mins should be sufficient for scheduling ~10k replicas
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
		waitForDesiredReplicasFunc(ctx, conn, out.GetNamespace(), out.GetName()))
	if err != nil {
		return diag.FromErr(err)
	}
	// We could wait for all pods to actually reach Ready state
	// but that means checking each pod status separately (which can be expensive at scale)
	// as there's no aggregate data available from the API

	log.Printf("[INFO] Submitted new replication controller: %#v", out)

	return resourceKubernetesReplicationControllerV1Read(ctx, d, meta)
}

func resourceKubernetesReplicationControllerV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesReplicationControllerV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading replication controller %s", name)
	rc, err := conn.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received replication controller: %#v", rc)

	err = d.Set("metadata", flattenMetadata(rc.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	spec, err := flattenReplicationControllerSpec(rc.Spec, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", spec)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesReplicationControllerV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		spec, err := expandReplicationControllerSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}

		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating replication controller %q: %v", name, string(data))
	out, err := conn.CoreV1().ReplicationControllers(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update replication controller: %s", err)
	}
	log.Printf("[INFO] Submitted updated replication controller: %#v", out)

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate),
		waitForDesiredReplicasFunc(ctx, conn, namespace, name))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceKubernetesReplicationControllerV1Read(ctx, d, meta)
}

func resourceKubernetesReplicationControllerV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
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
		return diag.FromErr(err)
	}
	_, err = conn.CoreV1().ReplicationControllers(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	// Wait until all replicas are gone
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete),
		waitForDesiredReplicasFunc(ctx, conn, namespace, name))
	if err != nil {
		return diag.FromErr(err)
	}

	err = conn.CoreV1().ReplicationControllers(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	// Wait for Delete to finish. Necessary for ForceNew operations.
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Replication Controller (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Replication controller %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesReplicationControllerV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking replication controller %s", name)
	_, err = conn.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func waitForDesiredReplicasFunc(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return func() *retry.RetryError {
		rc, err := conn.CoreV1().ReplicationControllers(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}

		desiredReplicas := *rc.Spec.Replicas
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			rc.GetName(), rc.Status.FullyLabeledReplicas, desiredReplicas)

		if rc.Status.FullyLabeledReplicas == desiredReplicas {
			return nil
		}

		return retry.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, rc.GetName(), rc.Status.FullyLabeledReplicas))
	}
}
