package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func dataSourceKubernetesMutatingWebhookConfiguration() *schema.Resource {
	apiDoc := admissionregistrationv1.MutatingWebhookConfiguration{}.SwaggerDoc()
	webhookDoc := admissionregistrationv1.MutatingWebhook{}.SwaggerDoc()
	return &schema.Resource{
		ReadContext: dataSourceKubernetesMutatingWebhookConfigurationRead,
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("mutating webhook configuration", false),
			"webhook": {
				Type:        schema.TypeList,
				Description: apiDoc["webhooks"],
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admission_review_versions": {
							Type:        schema.TypeList,
							Description: webhookDoc["admissionReviewVersions"],
							Computed:    true,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"client_config": {
							Type:        schema.TypeList,
							Description: webhookDoc["clientConfig"],
							Computed:    true,
							Elem: &schema.Resource{
								Schema: webhookClientConfigFields(),
							},
						},
						"failure_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["failurePolicy"],
							Computed:    true,
						},
						"match_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["matchPolicy"],
							Computed:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: webhookDoc["name"],
							Required:    true,
						},
						"namespace_selector": {
							Type:        schema.TypeList,
							Description: webhookDoc["namespaceSelector"],
							Computed:    true,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						"object_selector": {
							Type:        schema.TypeList,
							Description: webhookDoc["objectSelector"],
							Computed:    true,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						"reinvocation_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["reinvocationPolicy"],
							Computed:    true,
						},
						"rule": {
							Type:        schema.TypeList,
							Description: webhookDoc["rules"],
							Computed:    true,
							Elem: &schema.Resource{
								Schema: ruleWithOperationsFields(),
							},
						},
						"side_effects": {
							Type:        schema.TypeString,
							Description: webhookDoc["sideEffects"],
							Computed:    true,
							Optional:    true,
							ValidateFunc: validation.StringInSlice([]string{
								string(admissionregistrationv1.SideEffectClassUnknown),
								string(admissionregistrationv1.SideEffectClassNone),
								string(admissionregistrationv1.SideEffectClassSome),
								string(admissionregistrationv1.SideEffectClassNoneOnDryRun),
							}, false),
						},
						"timeout_seconds": {
							Type:        schema.TypeInt,
							Description: webhookDoc["timeoutSeconds"],
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesMutatingWebhookConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("metadata.0.name").(string)
	d.SetId(name)

	return resourceKubernetesMutatingWebhookConfigurationV1Read(ctx, d, meta)
}
