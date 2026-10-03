// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const daemonSetSDKv2ProviderAddressSuffix = "/hashicorp/kubernetes"

func (d *DaemonSetV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		namespace, name, err := kubernetes.IdParts(req.ID)
		if err != nil || namespace == "" || name == "" {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected a non-empty namespace/name ID, got %q.", req.ID))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s/%s", namespace, name))...)
		return
	}

	var identity common.NamespacedResourceIdentity
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if identity.Namespace.IsNull() || identity.Namespace.IsUnknown() || identity.Namespace.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid import identity", "Import identity must include a namespace.")
		return
	}
	if identity.Name.IsNull() || identity.Name.IsUnknown() || identity.Name.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid import identity", "Import identity must include a name.")
		return
	}
	if identity.APIVersion.IsNull() || identity.APIVersion.IsUnknown() || identity.APIVersion.ValueString() != daemonSetAPIVersion {
		resp.Diagnostics.AddError("Invalid import identity", fmt.Sprintf("Import identity api_version must be %q.", daemonSetAPIVersion))
		return
	}
	if identity.Kind.IsNull() || identity.Kind.IsUnknown() || identity.Kind.ValueString() != daemonSetKind {
		resp.Diagnostics.AddError("Invalid import identity", fmt.Sprintf("Import identity kind must be %q.", daemonSetKind))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx,
		path.Root("id"),
		fmt.Sprintf("%s/%s", identity.Namespace.ValueString(), identity.Name.ValueString()),
	)...)
}

func (d *DaemonSetV1) MoveState(ctx context.Context) []resource.StateMover {
	resourceSchema, schemaDiags := d.schemasForStateMoves(ctx)
	if schemaDiags.HasError() {
		return []resource.StateMover{{
			StateMover: func(_ context.Context, _ resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				resp.Diagnostics.Append(schemaDiags...)
			},
		}}
	}

	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "kubernetes_daemonset" ||
					(req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != 1) ||
					!strings.HasSuffix(req.SourceProviderAddress, daemonSetSDKv2ProviderAddressSuffix) {
					tflog.Debug(ctx, "MoveState: unsupported source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				rewrite := upgradeWorkloadState(req.SourceSchemaVersion, "strategy")
				value, err := common.DecodeLegacyState(ctx, req.SourceRawState, resourceSchema, rewrite)
				if err != nil {
					resp.Diagnostics.AddError("Unable to move daemon set state", err.Error())
					return
				}
				target := tfsdk.State{Schema: resourceSchema, Raw: value}
				resp.TargetState = target
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				var model DaemonSetV1Model
				resp.Diagnostics.Append(target.Get(ctx, &model)...)
				if resp.Diagnostics.HasError() {
					return
				}
				namespace, name, ok := daemonSetIdentityFields(model, &resp.Diagnostics)
				if !ok {
					return
				}
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, daemonSetIdentity(namespace, name))...)
			},
		},
	}
}

// UpgradeState accepts the SDKv2 schema versions 0 and 1.
func (d *DaemonSetV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	resourceSchema, schemaDiags := d.schemasForStateMoves(ctx)
	upgrader := func(rewrite func(map[string]any) error) resource.StateUpgrader {
		return resource.StateUpgrader{
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				if schemaDiags.HasError() {
					resp.Diagnostics.Append(schemaDiags...)
					return
				}
				value, err := common.DecodeLegacyState(ctx, req.RawState, resourceSchema, rewrite)
				if err != nil {
					resp.Diagnostics.AddError("Unable to upgrade daemon set state", err.Error())
					return
				}

				resp.State = tfsdk.State{Schema: resourceSchema, Raw: value}
			},
		}
	}
	return map[int64]resource.StateUpgrader{
		0: upgrader(upgradeWorkloadState(0, "strategy")),
		1: upgrader(upgradeWorkloadState(1, "strategy")),
	}
}

func (d *DaemonSetV1) schemasForStateMoves(ctx context.Context) (schema.Schema, diag.Diagnostics) {
	var diags diag.Diagnostics

	var schemaResp resource.SchemaResponse
	d.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	diags.Append(schemaResp.Diagnostics...)
	if diags.HasError() {
		return schema.Schema{}, diags
	}

	return schemaResp.Schema, diags
}

func daemonSetIdentityFields(model DaemonSetV1Model, diags *diag.Diagnostics) (string, string, bool) {
	if len(model.Metadata) != 1 {
		diags.AddError("Unable to set daemon set identity", fmt.Sprintf("Expected exactly 1 metadata element, got %d.", len(model.Metadata)))
		return "", "", false
	}
	namespace := model.Metadata[0].Namespace
	name := model.Metadata[0].Name
	if namespace.IsNull() || namespace.IsUnknown() || namespace.ValueString() == "" {
		diags.AddError("Unable to set daemon set identity", "Metadata namespace is empty or unknown.")
		return "", "", false
	}
	if name.IsNull() || name.IsUnknown() || name.ValueString() == "" {
		diags.AddError("Unable to set daemon set identity", "Metadata name is empty or unknown.")
		return "", "", false
	}
	return namespace.ValueString(), name.ValueString(), true
}
