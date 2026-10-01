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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podtemplate"
)

func (d *DeploymentV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A Deployment ensures that a specified number of pod replicas are running at any one time.",
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
			"metadata": common.NamespacedMetadataSchema("deployment", true),
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
					Optional: true,
					Computed: true,
					Default:  int64default.StaticInt64(0),
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"paused": schema.BoolAttribute{
					Optional: true,
					Computed: true,
					Default:  booldefault.StaticBool(false),
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"progress_deadline_seconds": schema.Int64Attribute{
					Optional: true,
					Computed: true,
					Default:  int64default.StaticInt64(600),
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"replicas": schema.StringAttribute{
					Optional: true,
					Computed: true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(regexp.MustCompile(`^$|^[+-]?\d+$`), "must be an integer string or empty"),
					},
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"revision_history_limit": schema.Int64Attribute{
					Optional: true,
					Computed: true,
					Default:  int64default.StaticInt64(10),
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"strategy": schema.ListNestedAttribute{
					Description: "The deployment strategy used to replace existing pods with new ones.",
					Optional:    true,
					Computed:    true,
					Validators: []validator.List{
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
								Validators:  []validator.List{listvalidator.SizeAtMost(1)},
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"max_surge": schema.StringAttribute{
											Description: "Maximum additional pods, as a nonnegative integer or percentage. Defaults to 25%.",
											Optional:    true,
											Computed:    true,
											Default:     stringdefault.StaticString("25%"),
											Validators:  []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^([0-9]+|[0-9]+%|)$`), "must be a nonnegative integer, percentage, or empty string")},
										},
										"max_unavailable": schema.StringAttribute{
											Description: "Maximum unavailable pods, as a nonnegative integer or percentage. Defaults to 25%.",
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
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		PlanModifiers: []planmodifier.List{
			listplanmodifier.RequiresReplace(),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"match_labels": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
					PlanModifiers: []planmodifier.Map{
						mapplanmodifier.RequiresReplace(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"match_expressions": schema.ListNestedBlock{
					PlanModifiers: []planmodifier.List{
						listplanmodifier.RequiresReplace(),
					},
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"key": schema.StringAttribute{
								Optional: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
								},
							},
							"operator": schema.StringAttribute{
								Optional: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
								},
							},
							"values": schema.SetAttribute{
								Optional:    true,
								ElementType: types.StringType,
								PlanModifiers: []planmodifier.Set{
									setplanmodifier.RequiresReplace(),
								},
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
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
			listvalidator.SizeAtMost(1),
			listvalidator.IsRequired(),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": templateMetadataBlock(),
				"spec": podtemplate.SpecBlock(podtemplate.Options{
					RestartPolicyAlways: true,
				}),
			},
		},
	}
}

func templateMetadataBlock() schema.ListNestedBlock {
	block := common.NamespacedMetadataSchema("pod", true)
	namespace := block.NestedObject.Attributes["namespace"].(schema.StringAttribute)
	namespace.Computed = false
	namespace.Default = nil
	namespace.PlanModifiers = []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	}
	block.NestedObject.Attributes["namespace"] = namespace
	return block
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
