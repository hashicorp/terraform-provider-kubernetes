// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	policy "k8s.io/api/policy/v1"
)

func (r *PodDisruptionBudgetV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	specDocs := policy.PodDisruptionBudgetSpec{}.SwaggerDoc()
	threshold := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Description: description,
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			Validators:  []validator.String{nullableIntOrPercentValidator{}},
		}
	}
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A Pod Disruption Budget limits the number of pods of a replicated application that are down simultaneously from voluntary disruptions. For example, a quorum-based application would like to ensure that the number of replicas running is never brought below the number needed for a quorum. A web front end might want to ensure that the number of replicas serving load never falls below a certain percentage of the total.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("pod disruption budget", true),
			"spec": schema.ListNestedBlock{
				Description:   policy.PodDisruptionBudget{}.SwaggerDoc()["spec"] + " Exactly one spec block is required.",
				Validators:    []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
				PlanModifiers: []planmodifier.List{podDisruptionBudgetSpecRequiresReplace{}},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"min_available":   threshold(specDocs["minAvailable"]),
						"max_unavailable": threshold(specDocs["maxUnavailable"]),

						"selector": schema.SingleNestedAttribute{
							Description: specDocs["selector"],
							Required:    true,
							Attributes: map[string]schema.Attribute{
								"match_labels": schema.MapAttribute{
									Description: "A map of label keys and values. The requirements are ANDed.",
									Optional:    true,
									ElementType: types.StringType,
								},
								"match_expressions": schema.ListNestedAttribute{
									Description: "A list of label selector requirements. The requirements are ANDed.",
									Optional:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"key": schema.StringAttribute{
												Description: "The label key that the selector applies to.",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString(""),
											},
											"operator": schema.StringAttribute{
												Description: "A key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString(""),
											},
											"values": schema.SetAttribute{
												Description: "An array of string values. Non-empty for In and NotIn; empty for Exists and DoesNotExist.",
												Optional:    true,
												ElementType: types.StringType,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

type nullableIntOrPercentValidator struct{}

func (nullableIntOrPercentValidator) Description(context.Context) string {
	return "Must be empty, a signed 32-bit integer, or a percentage between 0% and 100%."
}

func (v nullableIntOrPercentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v nullableIntOrPercentValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := validateNullableIntOrPercent(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid integer or percentage", err.Error())
	}
}

func validateNullableIntOrPercent(value string) error {
	if value == "" {
		return nil
	}
	number := strings.TrimSuffix(value, "%")
	n, err := strconv.ParseInt(number, 10, 32)
	if err != nil {
		return fmt.Errorf("Cannot parse %q as an integer or percentage: %s", value, err)
	}
	if strings.HasSuffix(value, "%") && (n < 0 || n > 100) {
		return fmt.Errorf("%q is not between 0%% and 100%%.", value)
	}
	return nil
}
