// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceKubernetesValidatingAdmissionPolicy() *schema.Resource {
	return &schema.Resource{
		Description:   "",
		CreateContext: resourceKubernetesValidatingAdmissionPolicyCreate,
		ReadContext:   resourceKubernetesValidatingAdmissionPolicyRead,
		UpdateContext: resourceKubernetesValidatingAdmissionPolicyUpdate,
		DeleteContext: resourceKubernetesValidatingAdmissionPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("validating admission policy", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Rule defining a set of permissions for the role",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"audit_annotations": {
							Type:        schema.TypeList,
							Description: "auditAnnotations contains CEL expressions which are used to produce audit annotations for the audit event of the API request.",
							Required:    true,
							Elem: &schema.Resource{
								Schema: auditAnnotationsFields(),
							},
						},
						"failure_policy": {
							Type:        schema.TypeString,
							Description: "failurePolicy defines how to handle failures for the admission policy.",
							Required:    true,
							Default:     "Fail",
							ValidateFunc: validation.StringInSlice([]string{
								"Fail",
								"Ignore",
							}, false),
						},
						"match_conditions": {
							Type:        schema.TypeList,
							Description: "MatchConditions is a list of conditions that must be met for a request to be validated.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: matchConditionsFields(),
							},
						},
						"match_constraints": {
							Type:        schema.TypeList,
							Description: "MatchConstraints specifies what resources this policy is designed to validate.",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: matchConstraintsFields(),
							},
						},
						"param_kind": {
							Type:        schema.TypeList,
							Description: "ParamKind specifies the kind of resources used to parameterize this policy",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: paramKindFields(),
							},
						},
						"validations": {
							Type:        schema.TypeList,
							Description: "Validations contain CEL expressions which is used to apply the validation.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: validationFields(),
							},
						},
						"variable": {
							Type:        schema.TypeList,
							Description: "Variables contain definitions of variables that can be used in composition of other expressions.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: variableFields(),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesValidatingAdmissionPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return resourceKubernetesValidatingAdmissionPolicyRead(ctx, d, meta)
}

func resourceKubernetesValidatingAdmissionPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return nil
}

func resourceKubernetesValidatingAdmissionPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return resourceKubernetesValidatingAdmissionPolicyRead(ctx, d, meta)
}

func resourceKubernetesValidatingAdmissionPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return nil
}

func resourceKubernetesValidatingAdmissionPolicyExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {

	return true, nil
}
