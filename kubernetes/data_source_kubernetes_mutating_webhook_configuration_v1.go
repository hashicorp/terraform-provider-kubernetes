// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesMutatingWebhookConfigurationV1() *schema.Resource {
	apiDoc := admissionregistrationv1.MutatingWebhookConfiguration{}.SwaggerDoc()
	webhookDoc := admissionregistrationv1.MutatingWebhook{}.SwaggerDoc()
	return &schema.Resource{
		ReadContext: dataSourceKubernetesMutatingWebhookConfigurationV1Read,
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

func dataSourceKubernetesMutatingWebhookConfigurationV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(metadata.Name)

	log.Printf("[INFO] Reading mutating webhook configuration %s", metadata.Name)
	cfg, err := conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received mutating webhook configuration: %#v", cfg)

	err = d.Set("metadata", flattenMetadataFields(cfg.ObjectMeta))
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Setting mutating webhook configuration to: %#v", cfg.Webhooks)

	err = d.Set("webhook", flattenMutatingWebhooks(cfg.Webhooks))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
