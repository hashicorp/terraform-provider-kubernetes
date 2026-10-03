// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
)

var deploymentFrozenSchema = common.FrozenSchema(buildDeploymentSchema)

func (d *DeploymentV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	deploymentFrozenSchema(ctx, req, resp)
}

func buildDeploymentSchema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 2,
		Description: "A Deployment ensures that a specified number of pod “replicas” are running at any one time. " +
			"In other words, a Deployment makes sure that a pod or homogeneous set of pods are always up and available. " +
			"If there are too many pods, it will kill some. If there are too few, the Deployment will start more.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wait_for_rollout": schema.BoolAttribute{
				Description: "Wait for the rollout of the deployment to complete. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("deployment", true)),
			"spec":     deploymentSpecBlock(),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
		},
	}
}

func deploymentSpecBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Spec defines the desired behavior of the deployment.",
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
			listvalidator.SizeAtMost(1),
			listvalidator.IsRequired(),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"min_ready_seconds": schema.Int64Attribute{
					Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"paused": schema.BoolAttribute{
					Description: "Indicates that the deployment is paused.",
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"progress_deadline_seconds": schema.Int64Attribute{
					Description: "The maximum time in seconds for a deployment to make progress before it is considered to be failed. The deployment controller will continue to process failed deployments and a condition with a ProgressDeadlineExceeded reason will be surfaced in the deployment status. Note that progress will not be estimated during the time a deployment is paused. Defaults to 600s.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(600),
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"replicas": schema.StringAttribute{
					Description: "Number of desired pods. This is a string to be able to distinguish between explicit zero and not specified.",
					Optional:    true,
					Computed:    true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(regexp.MustCompile(`^$|^[+-]?\d+$`), "must be an integer string or empty"),
					},
					PlanModifiers: []planmodifier.String{
						deploymentEmptyReplicasUseState{},
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"revision_history_limit": schema.Int64Attribute{
					Description: "The number of old ReplicaSets to retain to allow rollback. This is a pointer to distinguish between explicit zero and not specified. Defaults to 10.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(10),
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"strategy": schema.ListNestedAttribute{
					Description: "The deployment strategy used to replace existing pods with new ones.",
					Optional:    true,
					Computed:    true,
					PlanModifiers: []planmodifier.List{
						listplanmodifier.UseStateForUnknown(),
					},
					Validators: []validator.List{
						common.NotEmptyList(),
						listvalidator.SizeAtMost(1),
					},
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Type of deployment. Can be Recreate or RollingUpdate. Defaults to RollingUpdate.",
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("RollingUpdate"),
								Validators:  []validator.String{stringvalidator.OneOf("RollingUpdate", "Recreate")},
							},
							"rolling_update": schema.ListNestedAttribute{
								Description: "Rolling update parameters, used only with the RollingUpdate strategy.",
								Optional:    true,
								Computed:    true,
								Validators:  []validator.List{common.NotEmptyList(), listvalidator.SizeAtMost(1)},
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"max_surge": schema.StringAttribute{
											Description: "The maximum number of pods that can be scheduled above the desired number of pods. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up. Defaults to 25%. Example: when this is set to 30%, the new RC can be scaled up immediately when the rolling update starts, such that the total number of old and new pods do not exceed 130% of desired pods. Once old pods have been killed, new RC can be scaled up further, ensuring that total number of pods running at any time during the update is at most 130% of desired pods.",
											Optional:    true,
											Computed:    true,
											Default:     stringdefault.StaticString("25%"),
											Validators:  []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^([0-9]+|[0-9]+%|)$`), "must be a nonnegative integer, percentage, or empty string")},
										},
										"max_unavailable": schema.StringAttribute{
											Description: "The maximum number of pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). Absolute number is calculated from percentage by rounding down. This can not be 0 if MaxSurge is 0. Defaults to 25%. Example: when this is set to 30%, the old RC can be scaled down to 70% of desired pods immediately when the rolling update starts. Once new pods are ready, old RC can be scaled down further, followed by scaling up the new RC, ensuring that the total number of pods available at all times during the update is at least 70% of desired pods.",
											Optional:    true,
											Computed:    true,
											Default:     stringdefault.StaticString("25%"),
											Validators:  []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^([0-9]+|[0-9]+%|)$`), "must be a nonnegative integer, percentage, or empty string")},
										},
									},
								},
							},
						},
					},
				},
			},
			Blocks: map[string]schema.Block{
				"selector": selectorBlock(),
				"template": templateBlock(),
			},
		},
	}
}

func selectorBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "A label query over pods that should match the Replicas count.",
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		PlanModifiers: []planmodifier.List{
			workloadSelectorRequiresReplace(),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"match_labels": schema.MapAttribute{
					Description: common.LabelSelectorMatchLabelsDescription,
					Optional:    true,
					ElementType: types.StringType,
				},
			},
			Blocks: map[string]schema.Block{
				"match_expressions": schema.ListNestedBlock{
					Description: common.LabelSelectorMatchExpressionsDescription,
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"key": schema.StringAttribute{
								Description: common.LabelSelectorKeyDescription,
								Optional:    true,
							},
							"operator": schema.StringAttribute{
								Description: common.LabelSelectorOperatorDescription,
								Optional:    true,
							},
							"values": schema.SetAttribute{
								Description: common.LabelSelectorValuesDescription,
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
		},
	}
}

func templateBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Template describes the pods that will be created.",
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
			listvalidator.SizeAtMost(1),
			listvalidator.IsRequired(),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": workloadTemplateMetadataBlock(),
				"spec":     podspec.For(podspec.Deployment()).Spec,
			},
		},
	}
}

func deploymentStrategyObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type": types.StringType,
			"rolling_update": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"max_surge":       types.StringType,
						"max_unavailable": types.StringType,
					},
				},
			},
		},
	}

}

// Like SDKv2, an explicitly empty replicas string delegates scaling to the API.
type deploymentEmptyReplicasUseState struct{}

func (deploymentEmptyReplicasUseState) Description(context.Context) string {
	return "Retains the API replica count when configuration delegates scaling with an empty string."
}

func (modifier deploymentEmptyReplicasUseState) MarkdownDescription(ctx context.Context) string {
	return modifier.Description(ctx)
}

func (deploymentEmptyReplicasUseState) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "" ||
		req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}
