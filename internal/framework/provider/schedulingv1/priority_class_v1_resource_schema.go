// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *PriorityClassV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = PriorityClassV1Schema()
}

// PriorityClassV1Schema returns the Plugin Framework schema for PriorityClassV1.
// Exported so that unit tests can construct tfsdk.State values for the state upgrader.
func PriorityClassV1Schema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID for this terraform resource",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"value": schema.Int64Attribute{
				MarkdownDescription: "The value of this priority class. This is the actual priority that pods receive when they have the name of this class in their pod spec.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An arbitrary string that usually provides guidelines on when this priority class should be used.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"global_default": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this PriorityClass should be considered as the default priority for pods that do not have any priority class. Only one PriorityClass can be marked as `globalDefault`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"preemption_policy": schema.StringAttribute{
				MarkdownDescription: "PreemptionPolicy is the Policy for preempting pods with lower priority. One of `Never`, `PreemptLowerPriority`. Defaults to `PreemptLowerPriority` if unset.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("PreemptLowerPriority"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("Never", "PreemptLowerPriority"),
				},
			},
		},
		// metadata uses common.MetadataSchema — the framework equivalent of SDK v2's
		// metadataSchema("priority class", true). ListNestedBlock preserves the
		// metadata.0.* state path for full SDKv2 wire compatibility.
		Blocks: map[string]schema.Block{
			"metadata": common.MetadataSchema("priority class", true),
		},
	}
}
