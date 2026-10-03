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

				var rewrite func(map[string]any) error
				if req.SourceSchemaVersion == 0 {
					rewrite = upgradeDaemonSetV0State(ctx)
				}
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
		0: upgrader(upgradeDaemonSetV0State(ctx)),
		1: upgrader(nil),
	}
}

// upgradeDaemonSetV0State converts schema version 0 container resources.
func upgradeDaemonSetV0State(ctx context.Context) func(map[string]any) error {
	return func(raw map[string]any) error {
		if err := validateDaemonSetV0State(raw); err != nil {
			return err
		}
		kubernetes.UpgradeTemplatePodSpecWithResourcesFieldV0ForFramework(ctx, raw)
		return nil
	}
}

func validateDaemonSetV0State(raw map[string]interface{}) error {
	specs, ok := raw["spec"].([]interface{})
	if !ok || len(specs) == 0 {
		return nil
	}
	spec, ok := specs[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("stored spec element has unexpected type %T", specs[0])
	}
	templates, ok := spec["template"].([]interface{})
	if !ok || len(templates) == 0 {
		return nil
	}
	template, ok := templates[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("stored template element has unexpected type %T", templates[0])
	}
	podSpecs, ok := template["spec"].([]interface{})
	if !ok || len(podSpecs) == 0 {
		return nil
	}
	podSpec, ok := podSpecs[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("stored pod spec element has unexpected type %T", podSpecs[0])
	}
	for _, field := range []string{"container", "init_container"} {
		containers, ok := podSpec[field].([]interface{})
		if !ok {
			continue
		}
		for i, value := range containers {
			container, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("stored %s element %d has unexpected type %T", field, i, value)
			}
			resources, ok := container["resources"].([]interface{})
			if !ok || len(resources) == 0 {
				continue
			}
			resource, ok := resources[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("stored %s element %d resources has unexpected type %T", field, i, resources[0])
			}
			for _, resourceField := range []string{"requests", "limits"} {
				values, ok := resource[resourceField].([]interface{})
				if !ok || len(values) == 0 {
					continue
				}
				if _, ok := values[0].(map[string]interface{}); !ok {
					return fmt.Errorf("stored %s element %d resources.%s has unexpected type %T", field, i, resourceField, values[0])
				}
			}
		}
	}
	return nil
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
