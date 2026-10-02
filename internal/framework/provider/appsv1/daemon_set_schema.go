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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
)

var daemonSetRollingValuePattern = regexp.MustCompile(`^(\+?[0-9]+|[1-9][0-9]?%|100%)$`)

// The schema is built and frozen once per process; see common.FrozenSchema.
var daemonSetFrozenSchema = common.FrozenSchema(buildDaemonSetSchema)

func (d *DaemonSetV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	daemonSetFrozenSchema(ctx, req, resp)
}

func buildDaemonSetSchema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: "A DaemonSet ensures that all (or some) Nodes run a copy of a Pod. " +
			"As nodes are added to the cluster, Pods are added to them. As nodes are removed from the cluster, " +
			"those Pods are garbage collected. Deleting a DaemonSet will clean up the Pods it created.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wait_for_rollout": schema.BoolAttribute{
				Description: "Wait for the rollout of the daemonset to complete.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("daemonset", true)),
			"spec":     daemonSetSpecSchema(),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
		},
	}
}

func daemonSetSpecSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Spec defines the specification of the desired behavior of the daemonset.",
		Validators: []validator.List{
			listvalidator.IsRequired(),
			listvalidator.SizeAtLeast(1),
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"min_ready_seconds": schema.Int64Attribute{
					Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
				},
				"revision_history_limit": schema.Int64Attribute{
					Description: "The number of old DaemonSet revisions to retain to allow rollback.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(10),
				},
				"strategy": daemonSetStrategyAttribute(),
			},
			Blocks: map[string]schema.Block{
				"selector": daemonSetSelectorSchema(),
				"template": daemonSetTemplateSchema(),
			},
		},
	}
}

func daemonSetSelectorSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "A label query over pods that are managed by the DaemonSet.",
		PlanModifiers: []planmodifier.List{
			workloadSelectorRequiresReplace(),
		},
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"match_labels": schema.MapAttribute{
					Description: "A map of {key,value} pairs. The requirements are ANDed.",
					Optional:    true,
					ElementType: types.StringType,
				},
			},
			Blocks: map[string]schema.Block{
				"match_expressions": schema.ListNestedBlock{
					Description: "A list of label selector requirements. The requirements are ANDed.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"key": schema.StringAttribute{
								Optional: true,
							},
							"operator": schema.StringAttribute{
								Optional: true,
							},
							"values": schema.SetAttribute{
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

func daemonSetTemplateSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "An object that describes the pod that will be created.",
		Validators: []validator.List{
			listvalidator.IsRequired(),
			listvalidator.SizeAtLeast(1),
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": workloadTemplateMetadataBlock(),
				"spec":     podspec.For(podspec.DaemonSet()).Spec,
			},
		},
	}
}

func daemonSetStrategyAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: "The deployment strategy used to replace existing pods with new ones.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Optional: true,
					Computed: true,
					Default:  stringdefault.StaticString("RollingUpdate"),
					Validators: []validator.String{
						stringvalidator.OneOf("RollingUpdate", "OnDelete"),
					},
				},
				"rolling_update": schema.ListNestedAttribute{
					Description: "Rolling update config params. Present only if type is RollingUpdate.",
					Optional:    true,
					Computed:    true,
					Validators: []validator.List{
						listvalidator.SizeAtMost(1),
					},
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"max_surge": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Default:  stringdefault.StaticString("0"),
								Validators: []validator.String{
									stringvalidator.RegexMatches(daemonSetRollingValuePattern, "must be an absolute number or percentage"),
								},
							},
							"max_unavailable": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Default:  stringdefault.StaticString("1"),
								Validators: []validator.String{
									stringvalidator.RegexMatches(daemonSetRollingValuePattern, "must be an absolute number or percentage"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func daemonSetStrategyObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type": types.StringType,
			"rolling_update": types.ListType{
				ElemType: daemonSetRollingUpdateObjectType(),
			},
		},
	}
}

func daemonSetRollingUpdateObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"max_surge":       types.StringType,
			"max_unavailable": types.StringType,
		},
	}
}

func daemonSetSpecPath() path.Path {
	return path.Root("spec").AtListIndex(0)
}
