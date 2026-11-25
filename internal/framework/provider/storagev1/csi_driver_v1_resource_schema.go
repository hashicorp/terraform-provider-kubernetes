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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	storagev1api "k8s.io/api/storage/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// Schema implements [resource.Resource].
func (r *CSIDriverV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The [Container Storage Interface](https://kubernetes-csi.github.io/docs/introduction.html) (CSI) is a standard for exposing arbitrary block and file storage systems to containerized workloads on Container Orchestration Systems (COs) like Kubernetes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.MetadataSchema("csi driver", true),
			// Optional and at most one, as in SDKv2. A list block keeps both the `spec { }`
			// syntax and the one-element list that existing state stores.
			"spec": schema.ListNestedBlock{
				Description: "Spec of the CSIDriver. At most one spec block may be set.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"attach_required": schema.BoolAttribute{
							Description:   "Indicates if the CSI volume driver requires an attach operation",
							Required:      true,
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
						},
						// SDKv2 cannot tell unset from false, so omitting this meant false.
						// The default keeps that, instead of a diff against the API's false.
						"pod_info_on_mount": schema.BoolAttribute{
							Description: "Indicates that the CSI volume driver requires additional pod information (like podName, podUID, etc.) during mount operations",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						// Computed because the API defaults an omitted value to ["Persistent"].
						"volume_lifecycle_modes": schema.ListAttribute{
							Description: "Defines what kind of volumes this CSI volume driver supports",
							ElementType: types.StringType,
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.List{
								listvalidator.ValueStringsAre(stringvalidator.OneOf(
									string(storagev1api.VolumeLifecyclePersistent),
									string(storagev1api.VolumeLifecycleEphemeral),
								)),
							},
						},
					},
				},
			},
		},
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *CSIDriverV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
}
