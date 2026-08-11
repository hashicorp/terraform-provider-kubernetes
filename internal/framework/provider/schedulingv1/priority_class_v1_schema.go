// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PriorityClassV1Schema returns the Plugin Framework schema for PriorityClassV1.
// Exported so that unit tests can construct tfsdk.State values for the state upgrader.
func PriorityClassV1Schema() schema.Schema {
	return schema.Schema{
		Version:             1,
		MarkdownDescription: "A PriorityClass is a non-namespaced object that defines a mapping from a priority class name to the integer value of the priority.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID for this terraform resource",
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"annotations": schema.MapAttribute{
						MarkdownDescription: "An unstructured key value map stored with the priority class that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"generate_name": schema.StringAttribute{
						MarkdownDescription: "Prefix, used by the server, to generate a unique name ONLY IF the name field has not been provided. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency",
						Optional:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"generation": schema.Int64Attribute{
						MarkdownDescription: "A sequence number representing a specific generation of the desired state.",
						Computed:            true,
					},
					"labels": schema.MapAttribute{
						MarkdownDescription: "Map of string keys and values that can be used to organize and categorize (scope and select) the priority class. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the priority class, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"resource_version": schema.StringAttribute{
						MarkdownDescription: "An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
						Computed:            true,
					},
					"uid": schema.StringAttribute{
						MarkdownDescription: "The unique in time and space value for this priority class. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids",
						Computed:            true,
					},
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
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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
	}
}

func (r *PriorityClassV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = PriorityClassV1Schema()
}
