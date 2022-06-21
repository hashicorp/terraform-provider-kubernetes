package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/validators"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	podDisruptionBudgetSpecDoc               = api.PodDisruptionBudget{}.SwaggerDoc()["spec"]
	podDisruptionBudgetSpecMaxUnavailableDoc = api.PodDisruptionBudget{}.SwaggerDoc()["maxUnavailable"]
	podDisruptionBudgetSpecMinAvailableDoc   = api.PodDisruptionBudget{}.SwaggerDoc()["minAvailable"]
	podDisruptionBudgetSpecSelectorDoc       = api.PodDisruptionBudget{}.SwaggerDoc()["selector"]
)

func resourceKubernetesPodDisruptionBudget() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesPodDisruptionBudgetCreate,
		ReadContext:   resourceKubernetesPodDisruptionBudgetRead,
		UpdateContext: resourceKubernetesPodDisruptionBudgetUpdate,
		DeleteContext: resourceKubernetesPodDisruptionBudgetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": providermetav1.NamespacedMetadataSchema("pod disruption budget", true),
			// Updates to spec not allowed until Kubernetes dependencies are updated to
			// 1.13; have to delete and recreate until then
			// https://github.com/kubernetes/kubernetes/issues/45398
			"spec": {
				Type:        schema.TypeList,
				Description: podDisruptionBudgetSpecDoc,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_unavailable": {
							Type:         schema.TypeString,
							Description:  podDisruptionBudgetSpecMaxUnavailableDoc,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validators.ValidateTypeStringNullableIntOrPercent,
						},
						"min_available": {
							Type:         schema.TypeString,
							Description:  podDisruptionBudgetSpecMinAvailableDoc,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validators.ValidateTypeStringNullableIntOrPercent,
						},
						"selector": {
							Type:        schema.TypeList,
							Description: podDisruptionBudgetSpecSelectorDoc,
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

func resourceKubernetesPodDisruptionBudgetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := providermetav1.PatchMetadata("metadata.0.", "/metadata/", d)
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating pod disruption budget %s: %s", d.Id(), ops)
	out, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted updated pod disruption budget: %#v", out)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetRead(ctx, d, meta)
}

func resourceKubernetesPodDisruptionBudgetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := providermetav1.ExpandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodDisruptionBudgetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	pdb := api.PodDisruptionBudget{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new pod disruption budget: %#v", pdb)
	out, err := conn.PolicyV1beta1().PodDisruptionBudgets(metadata.Namespace).Create(ctx, &pdb, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new pod disruption budget: %#v", out)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetRead(ctx, d, meta)
}

func resourceKubernetesPodDisruptionBudgetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesPodDisruptionBudgetExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading pod disruption budget %s", name)
	pdb, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received pod disruption budget: %#v", pdb)
	err = d.Set("metadata", providermetav1.FlattenMetadata(pdb.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", flattenPodDisruptionBudgetSpec(pdb.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesPodDisruptionBudgetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting pod disruption budget %#v", name)
	err = conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Pod disruption budget %#v deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPodDisruptionBudgetExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking pod disruption budget %s", name)
	_, err = conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
