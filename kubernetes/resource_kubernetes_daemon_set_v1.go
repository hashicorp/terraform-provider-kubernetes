// Copyright IBM Corp. 2017, 2026
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
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func resourceKubernetesDaemonSetV1(deprecationMessage string) *schema.Resource {
	return &schema.Resource{
		Description:        "A DaemonSet ensures that all (or some) Nodes run a copy of a Pod. As nodes are added to the cluster, Pods are added to them. As nodes are removed from the cluster, those Pods are garbage collected. Deleting a DaemonSet will clean up the Pods it created.",
		CreateContext:      resourceKubernetesDaemonSetV1Create,
		ReadContext:        resourceKubernetesDaemonSetV1Read,
		DeprecationMessage: deprecationMessage,
		UpdateContext:      resourceKubernetesDaemonSetV1Update,
		DeleteContext:      resourceKubernetesDaemonSetV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceIdentityImportNamespaced,
		},
		Identity: resourceIdentitySchemaNamespaced(),
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceKubernetesDaemonSetV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceKubernetesDaemonSetUpgradeV0,
			},
		},
		SchemaVersion: 1,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: resourceKubernetesDaemonSetSchemaV1(),
	}
}

func resourceKubernetesDaemonSetSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("daemonset", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the specification of the desired behavior of the daemonset. More info: https://v1-9.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#daemonset-v1-apps",
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
					"revision_history_limit": {
						Type:        schema.TypeInt,
						Description: "The number of old ReplicaSets to retain to allow rollback. This is a pointer to distinguish between explicit zero and not specified. Defaults to 10.",
						Optional:    true,
						Default:     10,
					},
					"selector": {
						Type:        schema.TypeList,
						Description: "A label query over pods that are managed by the DaemonSet.",
						Optional:    true,
						ForceNew:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: labelSelectorFields(true),
						},
					},
					"strategy": {
						Type:        schema.TypeList,
						Optional:    true,
						Computed:    true,
						Description: "The deployment strategy used to replace existing pods with new ones.",
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"type": {
									Type:         schema.TypeString,
									Description:  "Type of deployment. Can be 'RollingUpdate' or 'OnDelete'. Default is RollingUpdate.",
									Optional:     true,
									Default:      "RollingUpdate",
									ValidateFunc: validation.StringInSlice([]string{"RollingUpdate", "OnDelete"}, false),
								},
								"rolling_update": {
									Type:        schema.TypeList,
									Description: "Rolling update config params. Present only if type = 'RollingUpdate'.",
									Optional:    true,
									Computed:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"max_surge": {
												Type:         schema.TypeString,
												Description:  "The maximum number of nodes with an existing available DaemonSet pod that can have an updated DaemonSet pod during during an update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up to a minimum of 1. Default value is 0. Example: when this is set to 30%, at most 30% of the total number of nodes that should be running the daemon pod (i.e. status.desiredNumberScheduled) can have their a new pod created before the old pod is marked as deleted. The update starts by launching new pods on 30% of nodes. Once an updated pod is available (Ready for at least minReadySeconds) the old DaemonSet pod on that node is marked deleted. If the old pod becomes unavailable for any reason Ready transitions to false, is evicted, or is drained) an updated pod is immediatedly created on that node without considering surge limits. Allowing surge implies the possibility that the resources consumed by the daemonset on any given node can double if the readiness check fails, and so resource intensive daemonsets should take into account that they may cause evictionsduring disruption.",
												Optional:     true,
												Default:      0,
												ValidateFunc: validation.StringMatch(regexp.MustCompile(`^(0|[1-9][0-9]*|[1-9][0-9]?%|100%)$`), ""),
											},
											"max_unavailable": {
												Type:         schema.TypeString,
												Description:  "The maximum number of DaemonSet pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of total number of DaemonSet pods at the start of the update (ex: 10%). Absolute number is calculated from percentage by rounding up. This cannot be 0 if MaxSurge is 0 Default value is 1. Example: when this is set to 30%, at most 30% of the total number of nodes that should be running the daemon pod (i.e. status.desiredNumberScheduled) can have their pods stopped for an update at any given time. The update starts by stopping at most 30% of those DaemonSet pods and then brings up new DaemonSet pods in their place. Once the new pods are available, it then proceeds onto other DaemonSet pods, thus ensuring that at least 70% of original number of DaemonSet pods are available at all times during the update.",
												Optional:     true,
												Default:      1,
												ValidateFunc: validation.StringMatch(regexp.MustCompile(`^(0|[1-9][0-9]*|[1-9][0-9]?%|100%)$`), ""),
											},
										},
									},
								},
							},
						},
					},
					"template": {
						Type:        schema.TypeList,
						Description: "An object that describes the pod that will be created. The DaemonSet will create exactly one copy of this pod on every node that matches the template's node selector (or on every node if no node selector is specified). More info: https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/#pod-template",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: podTemplateFields("daemon set"),
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

func resourceKubernetesDaemonSetV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandDaemonSetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	daemonset := appsv1.DaemonSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new daemonset: %#v", daemonset)

	out, err := conn.AppsV1().DaemonSets(metadata.Namespace).Create(ctx, &daemonset, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create daemonset: %s", err)
	}

	if d.Get("wait_for_rollout").(bool) {
		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
			waitForDaemonSetPodsFunc(ctx, conn, metadata.Namespace, metadata.Name))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[INFO] Submitted new daemonset: %#v", out)

	return resourceKubernetesDaemonSetV1Read(ctx, d, meta)
}

func resourceKubernetesDaemonSetV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		spec, err := expandDaemonSetSpec(d.Get("spec").([]interface{}))
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
	log.Printf("[INFO] Updating daemonset: %q", name)

	out, err := conn.AppsV1().DaemonSets(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update daemonset: %s", err)
	}
	log.Printf("[INFO] Submitted updated daemonset: %#v", out)

	if d.Get("wait_for_rollout").(bool) {
		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate),
			waitForDaemonSetPodsFunc(ctx, conn, namespace, name))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceKubernetesDaemonSetV1Read(ctx, d, meta)
}

func resourceKubernetesDaemonSetV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesDaemonSetV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading daemonset %s", name)
	daemonset, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received daemonset: %#v", daemonset)

	err = d.Set("metadata", flattenMetadata(daemonset.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	spec, err := flattenDaemonSetSpec(daemonset.Spec, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", spec)
	if err != nil {
		return diag.FromErr(err)
	}

	err = setResourceIdentityNamespaced(d, "apps/v1", "DaemonSet", namespace, name)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceKubernetesDaemonSetV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting daemonset: %#v", name)

	err = conn.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] DaemonSet %s deleted", name)

	return nil
}

func resourceKubernetesDaemonSetV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking daemonset %s", name)
	_, err = conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func waitForDaemonSetPodsFunc(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return func() *retry.RetryError {
		daemonSet, err := conn.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}

		desiredPods := daemonSet.Status.DesiredNumberScheduled

		if daemonSet.Generation > daemonSet.Status.ObservedGeneration {
			return retry.RetryableError(fmt.Errorf("waiting for rollout to start."))
		}

		if daemonSet.Generation == daemonSet.Status.ObservedGeneration {
			if daemonSet.Status.NumberReady == desiredPods {
				return nil
			}
			return retry.RetryableError(fmt.Errorf("waiting for rollout to finish: %d pods desired; %d pods ready",
				desiredPods, daemonSet.Status.NumberReady))
		}
		return retry.NonRetryableError(fmt.Errorf("observed generation %d is not expected to be greater than generation %d", daemonSet.Status.ObservedGeneration, daemonSet.Generation))
	}
}
