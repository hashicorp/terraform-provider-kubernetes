// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	podDisruptionBudgetV1SpecDoc               = policy.PodDisruptionBudget{}.SwaggerDoc()["spec"]
	podDisruptionBudgetV1SpecMaxUnavailableDoc = policy.PodDisruptionBudget{}.SwaggerDoc()["maxUnavailable"]
	podDisruptionBudgetV1SpecMinAvailableDoc   = policy.PodDisruptionBudget{}.SwaggerDoc()["minAvailable"]
	podDisruptionBudgetV1SpecSelectorDoc       = policy.PodDisruptionBudget{}.SwaggerDoc()["selector"]
)

func resourceKubernetesPodDisruptionBudgetV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesPodDisruptionBudgetV1Create,
		ReadContext:   resourceKubernetesPodDisruptionBudgetV1Read,
		UpdateContext: resourceKubernetesPodDisruptionBudgetV1Update,
		DeleteContext: resourceKubernetesPodDisruptionBudgetV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("pod disruption budget", true),
			// Updates to spec not allowed until Kubernetes dependencies are updated to
			// 1.13; have to delete and recreate until then
			// https://github.com/kubernetes/kubernetes/issues/45398
			"spec": {
				Type:        schema.TypeList,
				Description: podDisruptionBudgetV1SpecDoc,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_unavailable": {
							Type:         schema.TypeString,
							Description:  podDisruptionBudgetV1SpecMaxUnavailableDoc,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateTypeStringNullableIntOrPercent,
						},
						"min_available": {
							Type:         schema.TypeString,
							Description:  podDisruptionBudgetV1SpecMinAvailableDoc,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateTypeStringNullableIntOrPercent,
						},
						"selector": {
							Type:        schema.TypeList,
							Description: podDisruptionBudgetV1SpecSelectorDoc,
							Required:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(false),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPodDisruptionBudgetV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	log.Printf("[INFO] Updating pod disruption budget %s: %s", d.Id(), ops)
	out, err := conn.PolicyV1().PodDisruptionBudgets(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted updated pod disruption budget: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetV1Read(ctx, d, meta)
}

func resourceKubernetesPodDisruptionBudgetV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodDisruptionBudgetV1Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	pdb := policy.PodDisruptionBudget{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new pod disruption budget: %#v", pdb)
	out, err := conn.PolicyV1().PodDisruptionBudgets(metadata.Namespace).Create(ctx, &pdb, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new pod disruption budget: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetV1Read(ctx, d, meta)
}

func resourceKubernetesPodDisruptionBudgetV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesPodDisruptionBudgetV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading pod disruption budget %s", name)
	pdb, err := conn.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received pod disruption budget: %#v", pdb)
	err = d.Set("metadata", flattenMetadata(pdb.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", flattenPodDisruptionBudgetV1Spec(pdb.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesPodDisruptionBudgetV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting pod disruption budget %#v", name)
	err = conn.PolicyV1().PodDisruptionBudgets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Pod disruption budget %#v deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPodDisruptionBudgetV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking pod disruption budget %s", name)
	_, err = conn.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
