// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go#L93
	TimedOutReason = "ProgressDeadlineExceeded"
)

func resourceKubernetesDeploymentV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesDeploymentV1Create,
		ReadContext:   resourceKubernetesDeploymentV1Read,
		UpdateContext: resourceKubernetesDeploymentV1Update,
		DeleteContext: resourceKubernetesDeploymentV1Delete,
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
				Type:    resourceKubernetesDeploymentV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceKubernetesDeploymentUpgradeV0,
			},
		},
		SchemaVersion: 1,
		Schema:        resourceKubernetesDeploymentSchemaV1(),
	}
}

func resourceKubernetesDeploymentSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("deployment", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the specification of the desired behavior of the deployment. More info: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#deployment-v1-apps",
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
					"paused": {
						Type:        schema.TypeBool,
						Description: "Indicates that the deployment is paused.",
						Optional:    true,
						Default:     false,
					},
					"progress_deadline_seconds": {
						Type:        schema.TypeInt,
						Description: "The maximum time in seconds for a deployment to make progress before it is considered to be failed. The deployment controller will continue to process failed deployments and a condition with a ProgressDeadlineExceeded reason will be surfaced in the deployment status. Note that progress will not be estimated during the time a deployment is paused. Defaults to 600s.",
						Optional:    true,
						Default:     600,
					},
					"replicas": {
						Type:         schema.TypeString,
						Description:  "Number of desired pods. This is a string to be able to distinguish between explicit zero and not specified.",
						Optional:     true,
						Computed:     true,
						ValidateFunc: validateTypeStringNullableInt,
					},
					"revision_history_limit": {
						Type:        schema.TypeInt,
						Description: "The number of old ReplicaSets to retain to allow rollback. This is a pointer to distinguish between explicit zero and not specified. Defaults to 10.",
						Optional:    true,
						Default:     10,
					},
					"selector": {
						Type:        schema.TypeList,
						Description: "A label query over pods that should match the Replicas count.",
						Optional:    true,
						ForceNew:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"match_expressions": {
									Type:        schema.TypeList,
									Description: "A list of label selector requirements. The requirements are ANDed.",
									Optional:    true,
									ForceNew:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"key": {
												Type:        schema.TypeString,
												Description: "The label key that the selector applies to.",
												Optional:    true,
												ForceNew:    true,
											},
											"operator": {
												Type:        schema.TypeString,
												Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
												Optional:    true,
												ForceNew:    true,
											},
											"values": {
												Type:        schema.TypeSet,
												Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
												Optional:    true,
												ForceNew:    true,
												Elem:        &schema.Schema{Type: schema.TypeString},
												Set:         schema.HashString,
											},
										},
									},
								},
								"match_labels": {
									Type:        schema.TypeMap,
									Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
									Optional:    true,
									ForceNew:    true,
								},
							},
						},
					},
					"strategy": {
						Type:        schema.TypeList,
						Description: "The deployment strategy to use to replace existing pods with new ones.",
						Optional:    true,
						Computed:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"type": {
									Type:         schema.TypeString,
									Description:  "Type of deployment. Can be 'Recreate' or 'RollingUpdate'. Default is RollingUpdate.",
									Optional:     true,
									Default:      "RollingUpdate",
									ValidateFunc: validation.StringInSlice([]string{"RollingUpdate", "Recreate"}, false),
								},
								"rolling_update": {
									Type:        schema.TypeList,
									Description: "Rolling update config params. Present only if DeploymentStrategyType = RollingUpdate.",
									Optional:    true,
									Computed:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"max_surge": {
												Type:         schema.TypeString,
												Description:  "The maximum number of pods that can be scheduled above the desired number of pods. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up. Defaults to 25%. Example: when this is set to 30%, the new RC can be scaled up immediately when the rolling update starts, such that the total number of old and new pods do not exceed 130% of desired pods. Once old pods have been killed, new RC can be scaled up further, ensuring that total number of pods running at any time during the update is atmost 130% of desired pods.",
												Optional:     true,
												Default:      "25%",
												ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([0-9]+|[0-9]+%|)$`), ""),
											},
											"max_unavailable": {
												Type:         schema.TypeString,
												Description:  "The maximum number of pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). Absolute number is calculated from percentage by rounding down. This can not be 0 if MaxSurge is 0. Defaults to 25%. Example: when this is set to 30%, the old RC can be scaled down to 70% of desired pods immediately when the rolling update starts. Once new pods are ready, old RC can be scaled down further, followed by scaling up the new RC, ensuring that the total number of pods available at all times during the update is at least 70% of desired pods.",
												Optional:     true,
												Default:      "25%",
												ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([0-9]+|[0-9]+%|)$`), ""),
											},
										},
									},
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
							Schema: map[string]*schema.Schema{
								"metadata": namespacedMetadataSchemaIsTemplate("pod", true, true),
								"spec": {
									Type:        schema.TypeList,
									Description: "Spec defines the specification of the desired behavior of the deployment. More info: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#deployment-v1-apps",
									Required:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: podSpecFields(true, false),
									},
								},
							},
						},
					},
				},
			},
		},
		"wait_for_rollout": {
			Type:        schema.TypeBool,
			Description: "Wait for the rollout of the deployment to complete. Defaults to true.",
			Default:     true,
			Optional:    true,
		},
	}
}

func resourceKubernetesDeploymentV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandDeploymentSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new deployment: %#v", deployment)
	out, err := conn.AppsV1().Deployments(metadata.Namespace).Create(ctx, &deployment, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create deployment: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[DEBUG] Waiting for deployment %s to schedule %d replicas", d.Id(), *out.Spec.Replicas)

	if d.Get("wait_for_rollout").(bool) {
		log.Printf("[INFO] Waiting for deployment %s/%s to rollout", out.ObjectMeta.Namespace, out.ObjectMeta.Name)
		err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
			waitForDeploymentReplicasFunc(ctx, conn, out.GetNamespace(), out.GetName()))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[INFO] Submitted new deployment: %#v", out)

	return resourceKubernetesDeploymentV1Read(ctx, d, meta)
}

func resourceKubernetesDeploymentV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		spec, err := expandDeploymentSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}

		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}

	if d.HasChange("spec.0.strategy") {
		o, n := d.GetChange("spec.0.strategy.0.type")

		if o.(string) == "RollingUpdate" && n.(string) == "Recreate" {
			ops = append(ops, &RemoveOperation{
				Path: "/spec/strategy/rollingUpdate",
			})
		}
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating deployment %q: %v", name, string(data))
	out, err := conn.AppsV1().Deployments(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update deployment: %s", err)
	}
	log.Printf("[INFO] Submitted updated deployment: %#v", out)

	if d.Get("wait_for_rollout").(bool) {
		log.Printf("[INFO] Waiting for deployment %s/%s to rollout", out.ObjectMeta.Namespace, out.ObjectMeta.Name)
		err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate),
			waitForDeploymentReplicasFunc(ctx, conn, out.GetNamespace(), out.GetName()))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceKubernetesDeploymentV1Read(ctx, d, meta)
}

func resourceKubernetesDeploymentV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesDeploymentV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading deployment %s", name)
	deployment, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received deployment: %#v", deployment)

	err = d.Set("metadata", flattenMetadata(deployment.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	spec, err := flattenDeploymentSpec(deployment.Spec, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", spec)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesDeploymentV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting deployment: %#v", name)

	err = conn.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Deployment (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deployment %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesDeploymentV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking deployment %s", name)
	_, err = conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

// GetDeploymentConditionInternal returns the condition with the provided type.
// Borrowed from: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go#L135
func GetDeploymentCondition(status appsv1.DeploymentStatus, condType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func waitForDeploymentReplicasFunc(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return func() *retry.RetryError {
		// Query the deployment to get a status update.
		dply, err := conn.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}

		var specReplicas int32 = 1 // default, according to API docs
		if dply.Spec.Replicas != nil {
			specReplicas = *dply.Spec.Replicas
		}

		if dply.Generation > dply.Status.ObservedGeneration {
			return retry.RetryableError(fmt.Errorf("Waiting for rollout to start"))
		}

		if dply.Generation == dply.Status.ObservedGeneration {
			cond := GetDeploymentCondition(dply.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == TimedOutReason {
				return retry.NonRetryableError(fmt.Errorf("Deployment exceeded its progress deadline"))
			}

			if dply.Status.UpdatedReplicas < specReplicas {
				return retry.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d out of %d new replicas have been updated...", dply.Status.UpdatedReplicas, specReplicas))
			}

			if dply.Status.Replicas > dply.Status.UpdatedReplicas {
				return retry.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d old replicas are pending termination...", dply.Status.Replicas-dply.Status.UpdatedReplicas))
			}

			if dply.Status.Replicas > dply.Status.ReadyReplicas {
				return retry.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d replicas wanted; %d replicas Ready", dply.Status.Replicas, dply.Status.ReadyReplicas))
			}

			if dply.Status.AvailableReplicas < dply.Status.UpdatedReplicas {
				return retry.RetryableError(fmt.Errorf("Waiting for rollout to finish: %d of %d updated replicas are available...", dply.Status.AvailableReplicas, dply.Status.UpdatedReplicas))
			}
			return nil
		}

		return retry.NonRetryableError(fmt.Errorf("Observed generation %d is not expected to be greater than generation %d", dply.Status.ObservedGeneration, dply.Generation))
	}
}
