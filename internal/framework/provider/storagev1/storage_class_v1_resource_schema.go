// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *StorageClassV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = StorageClassV1Schema()
}

// StorageClassV1Schema returns the Plugin Framework schema for StorageClassV1.
// Exported so that unit tests can construct tfsdk.State values for the state mover.
func StorageClassV1Schema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Storage class is the foundation of dynamic provisioning, allowing cluster administrators to define abstractions for the underlying storage platform. Read more [here](https://kubernetes.io/blog/2017/03/dynamic-provisioning-and-storage-classes-kubernetes/).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID for this terraform resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// ── Required ────────────────────────────────────────────────────
			"storage_provisioner": schema.StringAttribute{
				MarkdownDescription: "Indicates the type of the provisioner.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// ── Optional + ForceNew (map) ────────────────────────────────────
			"parameters": schema.MapAttribute{
				MarkdownDescription: "The parameters for the provisioner that should create volumes of this storage class.",
				ElementType:         types.StringType,
				Optional:            true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},

			// ── Optional + mutable scalars ───────────────────────────────────
			"reclaim_policy": schema.StringAttribute{
				MarkdownDescription: "Indicates the reclaim policy. One of `Delete`, `Retain` or `Recycle`. Defaults to `Delete`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Delete"),
				Validators: []validator.String{
					stringvalidator.OneOf("Delete", "Retain", "Recycle"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow_volume_expansion": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the storage class allows volume expansion.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					// Not RequiresReplace — this field can be patched in-place.
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			// ── Optional + ForceNew scalars ──────────────────────────────────
			"volume_binding_mode": schema.StringAttribute{
				MarkdownDescription: "Indicates when volume binding and dynamic provisioning should occur. One of `Immediate` or `WaitForFirstConsumer`. Defaults to `Immediate`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Immediate"),
				Validators: []validator.String{
					stringvalidator.OneOf("Immediate", "WaitForFirstConsumer"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// ── Optional + ForceNew set ──────────────────────────────────────
			"mount_options": schema.SetAttribute{
				MarkdownDescription: "Persistent Volumes that are dynamically created by a storage class will have the mount options specified.",
				ElementType:         types.StringType,
				Optional:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
		},

		// ── Blocks ──────────────────────────────────────────────────────────
		Blocks: map[string]schema.Block{
			// metadata uses common.MetadataSchema — cluster-scoped, generatable name.
			// It produces a ListNestedBlock that preserves the SDKv2 state wire shape
			// (metadata.0.name etc.) without a StateUpgrader.
			"metadata": common.MetadataSchema("storage class", true),

			// allowed_topologies is MaxItems:1 in SDKv2. Modelled as a
			// ListNestedBlock with SizeBetween(0,1) to allow the block to be
			// omitted entirely. RequiresReplaceIfConfigured is applied at the
			// nested block attribute level so that any configured topology
			// change causes a replace, matching SDKv2 ForceNew behaviour.
			"allowed_topologies": schema.ListNestedBlock{
				MarkdownDescription: "Restrict the node topologies where volumes can be dynamically provisioned.",
				Validators: []validator.List{
					listvalidator.SizeBetween(0, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"match_label_expressions": schema.ListNestedBlock{
							MarkdownDescription: "A list of topology selector requirements by labels.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										MarkdownDescription: "The label key that the selector applies to.",
										Optional:            true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.RequiresReplaceIfConfigured(),
										},
									},
									"values": schema.SetAttribute{
										MarkdownDescription: "An array of string values. One value must match the label to be selected.",
										ElementType:         types.StringType,
										Optional:            true,
										PlanModifiers: []planmodifier.Set{
											setplanmodifier.RequiresReplaceIfConfigured(),
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
