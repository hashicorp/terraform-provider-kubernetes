// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// updatable controls PodSpec field modifiers. SDKv2's Job-level ForceNew fields
// also apply to the JobSpec embedded in a CronJob.
func jobSpecBlock(updatable bool) schema.ListNestedBlock {
	immutableInt := []planmodifier.Int64{jobSpecInt64RequiresReplace()}
	immutableString := []planmodifier.String{jobSpecStringRequiresReplace()}
	immutableObjectList := []planmodifier.List{workloadObjectListRequiresReplace()}

	selector := schema.ListNestedAttribute{
		Optional:      true,
		Computed:      true,
		Description:   "A label query selecting the pods owned by this job. Changes require replacement, including within a CronJob template.",
		Validators:    []validator.List{listvalidator.SizeAtMost(1)},
		PlanModifiers: immutableObjectList,
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"match_labels": schema.MapAttribute{
				Optional: true, ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{jobSpecMapRequiresReplace()},
			},
			"match_expressions": schema.ListNestedAttribute{
				Optional: true, PlanModifiers: immutableObjectList,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"key":      schema.StringAttribute{Optional: true, PlanModifiers: immutableString},
					"operator": schema.StringAttribute{Optional: true, PlanModifiers: immutableString},
					"values": schema.SetAttribute{
						Optional: true, ElementType: types.StringType,
						PlanModifiers: []planmodifier.Set{jobSpecSetRequiresReplace()},
					},
				}},
			},
		}},
	}
	policy := schema.ListNestedBlock{
		Description:   "Rules for handling pod failures. Rules are evaluated in order; unmatched failures count toward the job's backoff limit. Adding or removing this block requires replacement, including within a CronJob template.",
		Validators:    []validator.List{listvalidator.SizeAtMost(1)},
		PlanModifiers: immutableObjectList,
		NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				Validators: []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{Optional: true},
					},
					Blocks: map[string]schema.Block{
						"on_exit_codes": schema.ListNestedBlock{
							Validators: []validator.List{listvalidator.SizeAtMost(1)},
							NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
								"container_name": schema.StringAttribute{Optional: true},
								"operator":       schema.StringAttribute{Optional: true},
								"values": schema.ListAttribute{
									Required: true, ElementType: types.Int64Type,
									Validators: []validator.List{
										listvalidator.SizeBetween(1, 255),
										listvalidator.ValueInt64sAre(int64validator.NoneOf(0)),
									},
								},
							}},
						},
						"on_pod_condition": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
								"status": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("True")},
								"type":   schema.StringAttribute{Optional: true},
							}},
						},
					},
				},
			},
		}},
	}
	if !updatable {
		// Kubernetes Job policies are immutable. The SDK's cardinality-only
		// parent rule allowed ineffective in-place edits that Update ignored.
		policy.PlanModifiers = []planmodifier.List{jobSpecPolicyRequiresReplace()}
		policy.Description = "Rules for handling pod failures. Rules are evaluated in order; unmatched failures count toward the job's backoff limit. The policy is immutable for an existing Job, so changes require replacement."
	}

	return schema.ListNestedBlock{
		Description: "Specification of the job. Exactly one spec block is required.",
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"active_deadline_seconds": schema.Int64Attribute{
					Optional: true, Validators: []validator.Int64{int64validator.AtLeast(1)},
					Description: "Maximum time in seconds the job may be active.",
				},
				"backoff_limit": schema.Int64Attribute{
					Optional: true, Computed: true, Default: int64default.StaticInt64(6),
					Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"backoff_limit_per_index": schema.Int64Attribute{
					Description: "Maximum retries per index in an Indexed job. Omission uses the provider default of zero rather than unsetting the API field. Changes require replacement, including within a CronJob template.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(0), PlanModifiers: immutableInt,
					Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"completions": schema.Int64Attribute{
					Description: "Desired number of successfully finished pods. Changes require replacement, including within a CronJob template.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(1),
					PlanModifiers: immutableInt, Validators: []validator.Int64{int64validator.AtLeast(1)},
				},
				"completion_mode": schema.StringAttribute{
					Description: "Whether pod completions are tracked as Indexed or NonIndexed. Changes require replacement, including within a CronJob template.",
					Optional:    true, Computed: true,
					PlanModifiers: immutableString,
					Validators:    []validator.String{stringvalidator.OneOf("Indexed", "NonIndexed")},
				},
				"manual_selector": schema.BoolAttribute{
					Optional: true, Computed: true, Default: booldefault.StaticBool(false),
					Description: "Whether the caller controls pod labels and selectors.",
				},
				"max_failed_indexes": schema.Int64Attribute{
					Description: "Maximum number of failed indexes before an Indexed job is marked failed and its remaining pods are terminated. Requires backoff_limit_per_index. Omission uses the provider default of zero rather than unsetting the API field.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(0), Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"parallelism": schema.Int64Attribute{
					Optional: true, Computed: true, Default: int64default.StaticInt64(1),
					Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"selector": selector,
				"ttl_seconds_after_finished": schema.StringAttribute{
					Optional: true, Validators: []validator.String{jobTTLValidator{}},
					Description: "Seconds to retain a finished job. Zero allows immediate deletion; omission disables automatic deletion.",
				},
			},
			Blocks: map[string]schema.Block{
				"pod_failure_policy": policy,
				"template":           podTemplateBlock(updatable),
			},
		},
	}
}

func jobSpecType() types.ObjectType {
	return jobSpecValueField().typ.(types.ListType).ElemType.(types.ObjectType)
}

// The Job and CronJob spec blocks, and the value-field trees derived from them,
// depend only on the static schema but are needed on every RPC. Build and
// freeze them once per process; callers share them read-only. They are
// assigned in init because the schema's plan modifiers refer back to them.
var jobSpecValueField, cronJobSpecValueField func() valueField

func init() {
	jobSpecValueField = sync.OnceValue(func() valueField {
		return blockValueField(common.FreezeListNestedBlock(jobSpecBlock(false)))
	})
	cronJobSpecValueField = sync.OnceValue(func() valueField {
		return blockValueField(common.FreezeListNestedBlock(cronJobSpecBlock()))
	})
}

type jobTTLValidator struct{}

func (jobTTLValidator) Description(context.Context) string {
	return "Must be a non-negative integer encoded as a string."
}

func (v jobTTLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (jobTTLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value, err := strconv.ParseInt(req.ConfigValue.ValueString(), 10, 32)
	if err != nil || value < 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid job TTL",
			fmt.Sprintf("%q must be a non-negative 32-bit integer.", req.ConfigValue.ValueString()))
	}
}
