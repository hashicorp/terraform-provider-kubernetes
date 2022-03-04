package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesMutatingWebhookConfigurationV1() *schema.Resource {
	apiDoc := admissionregistrationv1.MutatingWebhookConfiguration{}.SwaggerDoc()
	webhookDoc := admissionregistrationv1.MutatingWebhook{}.SwaggerDoc()
	return &schema.Resource{
		CreateContext: resourceKubernetesMutatingWebhookConfigurationV1Create,
		ReadContext:   resourceKubernetesMutatingWebhookConfigurationV1Read,
		UpdateContext: resourceKubernetesMutatingWebhookConfigurationV1Update,
		DeleteContext: resourceKubernetesMutatingWebhookConfigurationV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("mutating webhook configuration", true),
			"webhook": {
				Type:        schema.TypeList,
				Description: apiDoc["webhooks"],
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admission_review_versions": {
							Type:        schema.TypeList,
							Description: webhookDoc["admissionReviewVersions"],
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"client_config": {
							Type:        schema.TypeList,
							Description: webhookDoc["clientConfig"],
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: webhookClientConfigFields(),
							},
						},
						"failure_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["failurePolicy"],
							Optional:    true,
							Default:     string(admissionregistrationv1.Fail),
							ValidateFunc: validation.StringInSlice([]string{
								string(admissionregistrationv1.Fail),
								string(admissionregistrationv1.Ignore),
							}, false),
						},
						"match_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["matchPolicy"],
							Optional:    true,
							Default:     string(admissionregistrationv1.Equivalent),
							ValidateFunc: validation.StringInSlice([]string{
								string(admissionregistrationv1.Equivalent),
								string(admissionregistrationv1.Exact),
							}, false),
						},
						"name": {
							Type:        schema.TypeString,
							Description: webhookDoc["name"],
							Required:    true,
						},
						"namespace_selector": {
							Type:        schema.TypeList,
							Description: webhookDoc["namespaceSelector"],
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						"object_selector": {
							Type:        schema.TypeList,
							Description: webhookDoc["objectSelector"],
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						"reinvocation_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["reinvocationPolicy"],
							Optional:    true,
							Default:     string(admissionregistrationv1.NeverReinvocationPolicy),
							ValidateFunc: validation.StringInSlice([]string{
								string(admissionregistrationv1.NeverReinvocationPolicy),
								string(admissionregistrationv1.IfNeededReinvocationPolicy),
							}, false),
						},
						"rule": {
							Type:        schema.TypeList,
							Description: webhookDoc["rules"],
							Optional:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: ruleWithOperationsFields(),
							},
						},
						"side_effects": {
							Type:        schema.TypeString,
							Description: webhookDoc["sideEffects"],
							Optional:    true,
							ValidateFunc: validation.StringInSlice([]string{
								string(admissionregistrationv1.SideEffectClassUnknown),
								string(admissionregistrationv1.SideEffectClassNone),
								string(admissionregistrationv1.SideEffectClassSome),
								string(admissionregistrationv1.SideEffectClassNoneOnDryRun),
							}, false),
						},
						"timeout_seconds": {
							Type:         schema.TypeInt,
							Description:  webhookDoc["timeoutSeconds"],
							Default:      10,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 30),
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesMutatingWebhookConfigurationV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	cfg := admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: expandMetadata(d.Get("metadata").([]interface{})),
		Webhooks:   expandMutatingWebhooks(d.Get("webhook").([]interface{})),
	}

	log.Printf("[INFO] Creating new MutatingWebhookConfiguration: %#v", cfg)

	res, err := conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, &cfg, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new MutatingWebhookConfiguration: %#v", res)

	d.SetId(res.Name)

	return resourceKubernetesMutatingWebhookConfigurationV1Read(ctx, d, meta)
}

func resourceKubernetesMutatingWebhookConfigurationV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesMutatingWebhookConfigurationV1Exists(ctx, d, meta)
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

	name := d.Id()
	cfg, err := conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("metadata", flattenMetadata(cfg.ObjectMeta, d))
	if err != nil {
		return nil
	}

	log.Printf("[DEBUG] Setting webhook to: %#v", cfg.Webhooks)

	err = d.Set("webhook", flattenMutatingWebhooks(cfg.Webhooks))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesMutatingWebhookConfigurationV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("webhook") {
		op := &ReplaceOperation{
			Path: "/webhooks",
		}
		patch := expandMutatingWebhooks(d.Get("webhook").([]interface{}))
		op.Value = patch
		ops = append(ops, op)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	name := d.Id()
	log.Printf("[INFO] Updating MutatingWebhookConfiguration %q: %v", name, string(data))

	res, err := conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update MutatingWebhookConfiguration: %s", err)
	}

	log.Printf("[INFO] Submitted updated MutatingWebhookConfiguration: %#v", res)

	return resourceKubernetesMutatingWebhookConfigurationV1Read(ctx, d, meta)
}

func resourceKubernetesMutatingWebhookConfigurationV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Deleting MutatingWebhookConfiguration: %#v", name)
	err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] MutatingWebhookConfiguration %#v is deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesMutatingWebhookConfigurationV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking MutatingWebhookConfiguration %s", name)

	_, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
