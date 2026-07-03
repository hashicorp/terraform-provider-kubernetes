// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestpatch

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *ManifestPatch) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Owns and reconciles a subset of fields on a **pre-existing** Kubernetes object " +
			"using Server-Side Apply, without creating or deleting the object. Use it to patch objects owned by " +
			"Helm, an operator, or a managed add-on (e.g. add an annotation to a Service, set env on the EKS " +
			"`aws-node` DaemonSet). Destroying the resource relinquishes the fields by default — it does not delete " +
			"the object. See RFC-012.",
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "apiVersion of the target object (e.g. \"apps/v1\").",
				Required:            true,
				PlanModifiers:       replace,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "Kind of the target object (e.g. \"Deployment\").",
				Required:            true,
				PlanModifiers:       replace,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "metadata.name of the target object.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "metadata.namespace of the target object. Omit for cluster-scoped kinds.",
				Optional:            true,
				PlanModifiers:       replace,
			},
			"patch": schema.DynamicAttribute{
				MarkdownDescription: "The fields to own, as an object (e.g. `{ spec = { replicas = 3 } }`). Applied " +
					"with Server-Side Apply, so this resource owns exactly these fields. Setting a leaf to `null` " +
					"**removes** that field. Exactly one of `patch` or `patch_json` must be set. Marked sensitive " +
					"because a patch may carry Secret data.",
				Optional:  true,
				Sensitive: true,
			},
			"patch_json": schema.StringAttribute{
				MarkdownDescription: "Escape hatch for operations the `patch` object can't express (RFC-6902 JSON " +
					"Patch, JSON merge patch, or strategic-merge patch). A JSON string interpreted per `patch_type`. " +
					"Exactly one of `patch` or `patch_json` must be set. Note: patch_json does not get owned-field " +
					"drift detection (no SSA ownership).",
				Optional:  true,
				Sensitive: true,
			},
			"patch_type": schema.StringAttribute{
				MarkdownDescription: "Patch strategy. With `patch`: `apply` (default, Server-Side Apply). With " +
					"`patch_json`: one of `json` (RFC-6902), `merge` (RFC-7386), or `strategic`.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(patchTypeApply, patchTypeMerge, patchTypeStrategic, patchTypeJSON),
				},
			},
			"ignore_fields": schema.ListAttribute{
				MarkdownDescription: "Dotted field paths (e.g. \"spec.replicas\") to exclude from drift detection, " +
					"so a controller that also writes an owned field (admission mutation, autoscaler) does not cause churn.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"field_manager": schema.StringAttribute{
				MarkdownDescription: "Server-Side Apply field manager name. Use a unique value so this patch co-owns " +
					"the object cleanly alongside other managers.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(defaultFieldManager),
			},
			"force_conflicts": schema.BoolAttribute{
				MarkdownDescription: "Force apply, taking ownership of patched fields managed by another field manager.",
				Optional:            true,
			},
			"destroy_behavior": schema.StringAttribute{
				MarkdownDescription: "What happens on destroy. `relinquish` (default) leaves the object untouched " +
					"(the patch is simply forgotten). `remove_fields` removes the fields this resource owns " +
					"(revert). `remove_fields` requires the SSA `patch` path.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(destroyRelinquish),
				Validators: []validator.String{
					stringvalidator.OneOf(destroyRelinquish, destroyRemoveFields),
				},
			},

			// Computed.
			"id": schema.StringAttribute{
				MarkdownDescription: "apiVersion=<>,kind=<>,namespace=<>,name=<>,fieldManager=<>",
				Computed:            true,
			},
			"owned_manifest": schema.StringAttribute{
				MarkdownDescription: "Canonical JSON of the fields owned by this resource's field manager (the drift " +
					"anchor). Empty for `patch_json` (no SSA ownership).",
				Computed:  true,
				Sensitive: true,
			},
			"object_exists": schema.BoolAttribute{
				MarkdownDescription: "Whether the target object currently exists.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// ConfigValidators enforces that exactly one patch surface is used.
func (r *ManifestPatch) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("patch"),
			path.MatchRoot("patch_json"),
		),
	}
}

// ValidateConfig enforces patch_type coherence with the chosen patch surface, and that
// remove_fields is only used with the SSA patch path.
func (r *ManifestPatch) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var m manifestPatchModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pt := m.PatchType.ValueString()

	if m.usesJSONPatch() {
		if pt == "" || pt == patchTypeApply {
			resp.Diagnostics.AddAttributeError(path.Root("patch_type"),
				"patch_type required for patch_json",
				"When using `patch_json`, set `patch_type` to one of \"json\", \"merge\", or \"strategic\".")
		}
		if m.DestroyBehavior.ValueString() == destroyRemoveFields {
			resp.Diagnostics.AddAttributeError(path.Root("destroy_behavior"),
				"remove_fields requires the SSA patch path",
				"`destroy_behavior = \"remove_fields\"` is only supported with the object `patch` (Server-Side "+
					"Apply), which tracks field ownership. With `patch_json`, use \"relinquish\".")
		}
		return
	}

	// Object patch path (SSA): patch_type must be empty or "apply".
	if pt != "" && pt != patchTypeApply {
		resp.Diagnostics.AddAttributeError(path.Root("patch_type"),
			"invalid patch_type for object patch",
			"When using the `patch` object, `patch_type` must be \"apply\" (the default). "+
				"Use `patch_json` for merge/strategic/json patches.")
	}
}
