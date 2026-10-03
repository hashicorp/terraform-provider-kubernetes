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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
)

// jobPodSpec is the pod spec of a Job's template, which Kubernetes does not
// allow to change, or of a CronJob's job template, which it does.
func jobPodSpec(job bool) *podspec.Built {
	if job {
		return podspec.For(podspec.Job())
	}
	return podspec.For(podspec.CronJob())
}

// jobSpecBlock is the JobSpec of a Job or of a CronJob's job template. Omitted
// scalars plan SDKv2's zero values, so state written by SDKv2 plans no change.
// The Job-level fields SDKv2 declared ForceNew replace a CronJob too.
func jobSpecBlock(job bool) schema.ListNestedBlock {
	policy := podFailurePolicyBlock()
	if job {
		// Kubernetes does not allow a Job's policy to change.
		policy.PlanModifiers = []planmodifier.List{jobPolicyRequiresReplace{}}
		policy.Description = "Rules for handling pod failures. Rules are evaluated in order; unmatched failures count toward the job's backoff limit. Kubernetes does not allow changing the policy of a Job, so any change replaces it."
	}
	return schema.ListNestedBlock{
		Description: "Specification of the job. Exactly one spec block is required.",
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"active_deadline_seconds": schema.Int64Attribute{
					Description: "Maximum time in seconds the job may be active. Zero means no deadline.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(0),
					Validators: []validator.Int64{int64validator.AtLeast(1)},
				},
				"backoff_limit": schema.Int64Attribute{
					Description: "Number of retries before the job is marked failed. Defaults to 6.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(6),
					Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"backoff_limit_per_index": schema.Int64Attribute{
					Description: "Maximum retries per index in an Indexed job. Applies only when completion_mode is Indexed, where omission sends zero. Changes require replacement.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(0),
					Validators:    []validator.Int64{int64validator.AtLeast(0)},
					PlanModifiers: []planmodifier.Int64{zeroEquivalentInt64RequiresReplace{}},
				},
				"completions": schema.Int64Attribute{
					Description: "Desired number of successfully finished pods. Defaults to 1. Changes require replacement.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(1),
					Validators:    []validator.Int64{int64validator.AtLeast(1)},
					PlanModifiers: []planmodifier.Int64{zeroEquivalentInt64RequiresReplace{}},
				},
				"completion_mode": schema.StringAttribute{
					Description: "Whether pod completions are tracked as `NonIndexed` (the Kubernetes default) or `Indexed`. Changes require replacement.",
					Optional:    true, Computed: true,
					Validators: []validator.String{stringvalidator.OneOf("Indexed", "NonIndexed")},
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						apiDefaultedStringRequiresReplace{},
					},
				},
				"manual_selector": schema.BoolAttribute{
					Description: "Controls generation of pod labels and pod selectors. Leave unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsible for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. More info: https://git.k8s.io/community/contributors/design-proposals/selector-generation.md",
					Optional:    true, Computed: true, Default: booldefault.StaticBool(false),
				},
				"max_failed_indexes": schema.Int64Attribute{
					Description: "Maximum number of failed indexes before an Indexed job is marked failed. Applies only when completion_mode is Indexed, where omission sends zero.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(0),
					Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"parallelism": schema.Int64Attribute{
					Description: "Maximum number of pods the job runs at any time. Defaults to 1.",
					Optional:    true, Computed: true, Default: int64default.StaticInt64(1),
					Validators: []validator.Int64{int64validator.AtLeast(0)},
				},
				"selector": jobSelectorAttribute(),
				"ttl_seconds_after_finished": schema.StringAttribute{
					Description: "Seconds to retain a finished job. Zero allows immediate deletion; omission disables automatic deletion.",
					Optional:    true, Computed: true, Default: stringdefault.StaticString(""),
					Validators: []validator.String{jobTTLValidator{}},
				},
			},
			Blocks: map[string]schema.Block{
				"pod_failure_policy": policy,
				"template": schema.ListNestedBlock{
					Description: "Describes the pod that will be created when executing a job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
					Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
					NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
						"metadata": templateMetadataBlock(common.MetadataSchema("job", true)),
						"spec":     jobPodSpec(job).Spec,
					}},
				},
			},
		},
	}
}

func jobSelectorAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "A label query over the pods owned by the job. Omit it to keep the selector Kubernetes generates. Changes require replacement.",
		Optional:    true, Computed: true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
			jobSelectorRequiresReplace{},
		},
		Attributes: map[string]schema.Attribute{
			"match_labels": schema.MapAttribute{Description: common.LabelSelectorMatchLabelsDescription, Optional: true, ElementType: types.StringType},
			"match_expressions": schema.ListNestedAttribute{
				Description: common.LabelSelectorMatchExpressionsDescription,
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"key":      schema.StringAttribute{Description: common.LabelSelectorKeyDescription, Optional: true},
					"operator": schema.StringAttribute{Description: common.LabelSelectorOperatorDescription, Optional: true},
					"values":   schema.SetAttribute{Description: common.LabelSelectorValuesDescription, Optional: true, ElementType: types.StringType},
				}},
			},
		},
	}
}

func podFailurePolicyBlock() schema.ListNestedBlock {
	emptyString := func() schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")}
	}
	return schema.ListNestedBlock{
		Description:   "Rules for handling pod failures. Rules are evaluated in order; unmatched failures count toward the job's backoff limit. Adding or removing the block requires replacement.",
		Validators:    []validator.List{listvalidator.SizeAtMost(1)},
		PlanModifiers: []planmodifier.List{listSizeRequiresReplace{}},
		NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				Description: "A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining rules are ignored.",
				Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{"action": emptyString()},
					Blocks: map[string]schema.Block{
						"on_exit_codes": schema.ListNestedBlock{
							Validators: []validator.List{listvalidator.SizeAtMost(1)},
							NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
								"container_name": emptyString(),
								"operator":       emptyString(),
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
								"type":   emptyString(),
							}},
						},
					},
				},
			},
		}},
	}
}

// templateMetadataBlock adapts object metadata to a pod or job template, as
// for the apps/v1 workloads. SDKv2 declared its name, generate_name and
// namespace ForceNew; the server never sets its other fields.
func templateMetadataBlock(block schema.ListNestedBlock) schema.ListNestedBlock {
	block = common.WithEmptyMetadataCompatibility(block)
	attributes := block.NestedObject.Attributes
	generateName := attributes["generate_name"].(schema.StringAttribute)
	generateName.PlanModifiers = []planmodifier.String{zeroEquivalentStringRequiresReplace{}}
	attributes["generate_name"] = generateName
	for _, name := range []string{"resource_version", "uid"} {
		a := attributes[name].(schema.StringAttribute)
		a.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
		attributes[name] = a
	}
	generation := attributes["generation"].(schema.Int64Attribute)
	generation.PlanModifiers = []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	attributes["generation"] = generation
	return block
}

// The JobSpec types of a Job and of a CronJob's job template, derived from the
// frozen schema once per process.
var (
	jobSpecType         = sync.OnceValue(func() types.ObjectType { return jobSpecObjectType(true) })
	jobTemplateSpecType = sync.OnceValue(func() types.ObjectType { return jobSpecObjectType(false) })
)

func jobSpecObjectType(job bool) types.ObjectType {
	return common.FreezeListNestedBlock(jobSpecBlock(job)).NestedObject.Type().(types.ObjectType)
}

func jobSpecTypeFor(job bool) types.ObjectType {
	if job {
		return jobSpecType()
	}
	return jobTemplateSpecType()
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
