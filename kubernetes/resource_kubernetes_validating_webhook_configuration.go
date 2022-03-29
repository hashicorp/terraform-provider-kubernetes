package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	copier "github.com/jinzhu/copier"
)

func resourceKubernetesValidatingWebhookConfiguration() *schema.Resource {
	apiDoc := admissionregistrationv1.ValidatingWebhookConfiguration{}.SwaggerDoc()
	webhookDoc := admissionregistrationv1.ValidatingWebhook{}.SwaggerDoc()
	return &schema.Resource{
		CreateContext: resourceKubernetesValidatingWebhookConfigurationCreate,
		ReadContext:   resourceKubernetesValidatingWebhookConfigurationRead,
		UpdateContext: resourceKubernetesValidatingWebhookConfigurationUpdate,
		DeleteContext: resourceKubernetesValidatingWebhookConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("validating webhook configuration", true),
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
						"rule": {
							Type:        schema.TypeList,
							Description: webhookDoc["rules"],
							Optional:    true,
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

func resourceKubernetesValidatingWebhookConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	cfg := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: expandMetadata(d.Get("metadata").([]interface{})),
		Webhooks:   expandValidatingWebhooks(d.Get("webhook").([]interface{})),
	}

	log.Printf("[INFO] Creating new ValidatingWebhookConfiguration: %#v", cfg)

	res := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return diag.FromErr(err)
	}
	if useadmissionregistrationv1beta1 {
		requestv1beta1 := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		responsev1beta1 := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		copier.Copy(requestv1beta1, cfg)
		responsev1beta1, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(ctx, requestv1beta1, metav1.CreateOptions{})
		copier.Copy(res, responsev1beta1)
	} else {
		res, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, &cfg, metav1.CreateOptions{})
	}

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new ValidatingWebhookConfiguration: %#v", res)

	d.SetId(res.Name)

	return resourceKubernetesValidatingWebhookConfigurationRead(ctx, d, meta)
}

func resourceKubernetesValidatingWebhookConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesValidatingWebhookConfigurationExists(ctx, d, meta)
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

	cfg := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	log.Printf("[INFO] Reading ValidatingWebhookConfiguration %s", name)
	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return diag.FromErr(err)
	}
	if useadmissionregistrationv1beta1 {
		cfgv1beta1 := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		cfgv1beta1, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		copier.Copy(cfg, cfgv1beta1)
	} else {
		cfg, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("metadata", flattenMetadata(cfg.ObjectMeta, d))
	if err != nil {
		return nil
	}

	log.Printf("[DEBUG] Setting webhook to: %#v", cfg.Webhooks)

	err = d.Set("webhook", flattenValidatingWebhooks(cfg.Webhooks))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesValidatingWebhookConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("webhook") {
		op := &ReplaceOperation{
			Path: "/webhooks",
		}

		patch := expandValidatingWebhooks(d.Get("webhook").([]interface{}))

		useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
		if err != nil {
			return diag.FromErr(err)
		}
		if useadmissionregistrationv1beta1 {
			patchv1beta1 := []admissionregistrationv1beta1.ValidatingWebhook{}
			copier.Copy(&patchv1beta1, &patch)
			op.Value = patchv1beta1
		} else {
			op.Value = patch
		}

		ops = append(ops, op)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	name := d.Id()
	log.Printf("[INFO] Updating ValidatingWebhookCoßnfiguration %q: %v", name, string(data))

	res := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return diag.FromErr(err)
	}
	if useadmissionregistrationv1beta1 {
		responsev1beta1 := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		responsev1beta1, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
		copier.Copy(res, responsev1beta1)
	} else {
		res, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	}
	if err != nil {
		return diag.Errorf("Failed to update ValidatingWebhookConfiguration: %s", err)
	}

	log.Printf("[INFO] Submitted updated ValidatingWebhookConfiguration: %#v", res)

	return resourceKubernetesValidatingWebhookConfigurationRead(ctx, d, meta)
}

func resourceKubernetesValidatingWebhookConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Deleting ValidatingWebhookConfiguration: %#v", name)
	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return diag.FromErr(err)
	}
	if useadmissionregistrationv1beta1 {
		err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(ctx, name, metav1.DeleteOptions{})
	} else {
		err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, name, metav1.DeleteOptions{})
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] ValidatingWebhookConfiguration %#v is deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesValidatingWebhookConfigurationExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking ValidatingWebhookConfiguration %s", name)

	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return false, err
	}
	if useadmissionregistrationv1beta1 {
		_, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	} else {
		_, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
