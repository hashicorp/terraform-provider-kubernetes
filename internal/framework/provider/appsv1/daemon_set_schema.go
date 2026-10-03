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
		Version: 2,
		Description: "A DaemonSet ensures that all (or some) Nodes run a copy of a Pod. " +
			"As nodes are added to the cluster, Pods are added to them. As nodes are removed from the cluster, " +
			"those Pods are garbage collected. Deleting a DaemonSet will clean up the Pods it created.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wait_for_rollout": schema.BoolAttribute{
				Description: "Wait for the rollout of the daemonset to complete. Defaults to true.",
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
					Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
				},
				"revision_history_limit": schema.Int64Attribute{
					Description: "The number of old DaemonSet revisions to retain to allow rollback. Defaults to 10.",
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

func daemonSetTemplateSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "An object that describes the pod that will be created. The DaemonSet will create exactly one copy of this pod on every node that matches the template's node selector (or on every node if no node selector is specified). More info: https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/#pod-template",
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
			common.NotEmptyList(),
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Description: "Type of DaemonSet update. Can be RollingUpdate or OnDelete. Defaults to RollingUpdate.",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("RollingUpdate"),
					Validators: []validator.String{
						stringvalidator.OneOf("RollingUpdate", "OnDelete"),
					},
				},
				"rolling_update": schema.ListNestedAttribute{
					Description: "Rolling update config params. Present only if type is RollingUpdate.",
					Optional:    true,
					Computed:    true,
					Validators: []validator.List{
						common.NotEmptyList(),
						listvalidator.SizeAtMost(1),
					},
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"max_surge": schema.StringAttribute{
								Description: "The maximum number of nodes with an existing available DaemonSet pod that can have an updated DaemonSet pod during an update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up to a minimum of 1. Default value is 0. Example: when this is set to 30%, at most 30% of the total number of nodes that should be running the daemon pod (i.e. status.desiredNumberScheduled) can have a new pod created before the old pod is marked as deleted. The update starts by launching new pods on 30% of nodes. Once an updated pod is available (Ready for at least minReadySeconds) the old DaemonSet pod on that node is marked deleted. If the old pod becomes unavailable for any reason (Ready transitions to false, is evicted, or is drained) an updated pod is immediately created on that node without considering surge limits. Allowing surge implies the possibility that the resources consumed by the daemonset on any given node can double if the readiness check fails, and so resource intensive daemonsets should take into account that they may cause evictions during disruption.",
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("0"),
								Validators: []validator.String{
									stringvalidator.RegexMatches(daemonSetRollingValuePattern, "must be an absolute number or percentage"),
								},
							},
							"max_unavailable": schema.StringAttribute{
								Description: "The maximum number of DaemonSet pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of total number of DaemonSet pods at the start of the update (ex: 10%). Absolute number is calculated from percentage by rounding up. This cannot be 0 if MaxSurge is 0. Default value is 1. Example: when this is set to 30%, at most 30% of the total number of nodes that should be running the daemon pod (i.e. status.desiredNumberScheduled) can have their pods stopped for an update at any given time. The update starts by stopping at most 30% of those DaemonSet pods and then brings up new DaemonSet pods in their place. Once the new pods are available, it then proceeds onto other DaemonSet pods, thus ensuring that at least 70% of original number of DaemonSet pods are available at all times during the update.",
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("1"),
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
