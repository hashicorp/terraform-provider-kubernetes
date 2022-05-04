package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesHorizontalPodAutoscalerV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesHorizontalPodAutoscalerV2Create,
		ReadContext:   resourceKubernetesHorizontalPodAutoscalerV2Read,
		UpdateContext: resourceKubernetesHorizontalPodAutoscalerV2Update,
		DeleteContext: resourceKubernetesHorizontalPodAutoscalerV2Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("horizontal pod autoscaler", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Behaviour of the autoscaler. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_replicas": {
							Type:        schema.TypeInt,
							Description: "Upper limit for the number of pods that can be set by the autoscaler.",
							Required:    true,
						},
						"metric": {
							Type:        schema.TypeList,
							Computed:    true,
							Optional:    true,
							Description: "The specifications for which to use to calculate the desired replica count (the maximum replica count across all metrics will be used). The desired replica count is calculated multiplying the ratio between the target value and the current value by the current number of pods. Ergo, metrics used must decrease as the pod count is increased, and vice-versa. See the individual metric source types for more information about how each type of metric must respond. If not set, the default metric will be set to 80% average CPU utilization.",
							Elem:        metricSpecFields(),
						},
						"min_replicas": {
							Type:        schema.TypeInt,
							Description: "Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.",
							Optional:    true,
							Default:     1,
						},
						"behavior": {
							Type:        schema.TypeList,
							Description: "Behavior configures the scaling behavior of the target in both Up and Down directions (scale_up and scale_down fields respectively).",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"scale_up": {
										Type:        schema.TypeList,
										Description: "Scaling policy for scaling Up",
										Optional:    true,
										Elem:        scalingRulesSpecFields(),
									},
									"scale_down": {
										Type:        schema.TypeList,
										Description: "Scaling policy for scaling Down",
										Optional:    true,
										Elem:        scalingRulesSpecFields(),
									},
								},
							},
						},
						"scale_target_ref": {
							Type:        schema.TypeList,
							Description: "Reference to scaled resource. e.g. Replication Controller",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"api_version": {
										Type:        schema.TypeString,
										Description: "API version of the referent",
										Optional:    true,
									},
									"kind": {
										Type:        schema.TypeString,
										Description: "Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
										Required:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
										Required:    true,
									},
								},
							},
						},
						"target_cpu_utilization_percentage": {
							Type:        schema.TypeInt,
							Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesHorizontalPodAutoscalerV2Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandHorizontalPodAutoscalerV2Spec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	hpa := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new horizontal pod autoscaler: %#v", hpa)
	out, err := conn.AutoscalingV2().HorizontalPodAutoscalers(metadata.Namespace).Create(ctx, &hpa, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerV2Read(ctx, d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerV2Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesHorizontalPodAutoscalerV2Exists(ctx, d, meta)
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
	log.Printf("[INFO] Reading horizontal pod autoscaler %s", name)
	hpa, err := conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received horizontal pod autoscaler: %#v", hpa)
	err = d.Set("metadata", flattenMetadata(hpa.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenHorizontalPodAutoscalerV2Spec(hpa.Spec)
	log.Printf("[DEBUG] Flattened horizontal pod autoscaler spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesHorizontalPodAutoscalerV2Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		diffOps := patchHorizontalPodAutoscalerV2Spec("spec.0.", "/spec", d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating horizontal pod autoscaler %q: %v", name, string(data))
	out, err := conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update horizontal pod autoscaler: %s", err)
	}
	log.Printf("[INFO] Submitted updated horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerV2Read(ctx, d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerV2Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleting horizontal pod autoscaler: %#v", name)
	err = conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Horizontal Pod Autoscaler %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesHorizontalPodAutoscalerV2Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking horizontal pod autoscaler %s", name)
	_, err = conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
