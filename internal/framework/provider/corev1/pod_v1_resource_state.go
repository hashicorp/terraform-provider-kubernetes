// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

const (
	podV1ImportStateErrSummary  = "Unable to import kubernetes_pod_v1"
	podV1MoveStateErrSummary    = "Unable to move kubernetes_pod state"
	podV1UpgradeStateErrSummary = "Unable to upgrade kubernetes_pod_v1 state"
	podUnversionedTypeName      = "kubernetes_pod"
)

// ImportState implements [resource.ResourceWithImportState].
func (p *PodV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		namespace, name, err := podV1ParseID(req.ID)
		if err != nil {
			resp.Diagnostics.AddError(podV1ImportStateErrSummary, err.Error())
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
		// This provider-only default cannot be recovered from the Kubernetes API.
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_state"), types.ListValueMust(types.StringType, nil))...)
		if resp.Diagnostics.HasError() || resp.Identity == nil {
			return
		}

		resp.Diagnostics.Append(resp.Identity.Set(ctx, podV1Identity(namespace, name))...)
		return
	}

	if req.Identity == nil {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary, "Import requires either an id or identity.")
		return
	}

	var name, namespace types.String
	var apiVersion, kind types.String
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("name"), &name)...)
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("namespace"), &namespace)...)
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("api_version"), &apiVersion)...)
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("kind"), &kind)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if name.IsNull() || name.IsUnknown() || name.ValueString() == "" {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary, "Identity must include a non-empty name.")
		return
	}
	if apiVersion.IsUnknown() || apiVersion.IsNull() || apiVersion.ValueString() == "" {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary, "Identity must include api_version.")
		return
	}
	if apiVersion.ValueString() != podAPIVersion {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary,
			fmt.Sprintf("Identity api_version %q is invalid. Expected %q.", apiVersion.ValueString(), podAPIVersion))
		return
	}
	if kind.IsUnknown() || kind.IsNull() || kind.ValueString() == "" {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary, "Identity must include kind.")
		return
	}
	if kind.ValueString() != podKind {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary,
			fmt.Sprintf("Identity kind %q is invalid. Expected %q.", kind.ValueString(), podKind))
		return
	}
	if namespace.IsUnknown() {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary, "Identity namespace cannot be unknown.")
		return
	}

	ns := corev1.NamespaceDefault
	if !namespace.IsNull() {
		ns = namespace.ValueString()
		if ns == "" {
			resp.Diagnostics.AddError(podV1ImportStateErrSummary, "Identity namespace cannot be empty.")
			return
		}
	}
	id := fmt.Sprintf("%s/%s", ns, name.ValueString())
	if _, _, err := podV1ParseID(id); err != nil {
		resp.Diagnostics.AddError(podV1ImportStateErrSummary, err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_state"), types.ListValueMust(types.StringType, nil))...)
	if resp.Diagnostics.HasError() || resp.Identity == nil {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, podV1Identity(ns, name.ValueString()))...)
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
func (p *PodV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(podKind, podAPIVersion)
}

// UpgradeState implements [resource.ResourceWithUpgradeState] for the SDKv2
// schema versions 0 and 1.
func (p *PodV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	upgrader := func(version int64) resource.StateUpgrader {
		return resource.StateUpgrader{
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				state, _, _, diags := p.decodeLegacyState(ctx, req.RawState, version, podV1UpgradeStateErrSummary)
				resp.Diagnostics.Append(diags...)
				if !resp.Diagnostics.HasError() {
					resp.State = state
				}
			},
		}
	}
	return map[int64]resource.StateUpgrader{0: upgrader(0), 1: upgrader(1)}
}

// MoveState moves kubernetes_pod state to kubernetes_pod_v1. It does not guard
// on SourceIdentitySchemaVersion, which terraform-plugin-go v0.29.0 leaves at zero.
func (p *PodV1) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != podUnversionedTypeName ||
					(req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != 1) ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					tflog.Debug(ctx, "MoveState: not a kubernetes_pod v0/v1 source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				state, namespace, name, diags := p.decodeLegacyState(ctx, req.SourceRawState, req.SourceSchemaVersion, podV1MoveStateErrSummary)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(podV1ValidateSourceIdentity(req.SourceIdentity, namespace, name)...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.TargetState = state
				if resp.TargetIdentity == nil {
					return
				}
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, podV1Identity(namespace, name))...)
			},
		},
	}
}

func (p *PodV1) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func podV1Identity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(podAPIVersion),
			Kind:       types.StringValue(podKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

type sdkv2PodSourceIdentity struct {
	APIVersion *string `json:"api_version"`
	Kind       *string `json:"kind"`
	Name       *string `json:"name"`
	Namespace  *string `json:"namespace"`
}

// decodeLegacyState decodes state written by the SDKv2 kubernetes_pod or
// kubernetes_pod_v1 at the given schema version.
func (p *PodV1) decodeLegacyState(ctx context.Context, raw *tfprotov6.RawState, version int64, summary string) (tfsdk.State, string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var schemaResp resource.SchemaResponse
	p.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	diags.Append(schemaResp.Diagnostics...)
	state := tfsdk.State{Schema: schemaResp.Schema}

	var namespace, name string
	value, err := common.DecodeLegacyState(ctx, raw, schemaResp.Schema, func(values map[string]any) error {
		var err error
		if namespace, name, err = common.LegacyStateName(values); err != nil {
			return err
		}
		// Normalise a "" timeout, which SDKv2 accepted, to null.
		if timeouts, ok := values["timeouts"].(map[string]any); ok {
			for key, timeout := range timeouts {
				if timeout == "" {
					timeouts[key] = nil
				}
			}
		}
		// SDKv2 left an unset target_state null; the schema defaults it to empty.
		if values["target_state"] == nil {
			values["target_state"] = []any{}
		}
		if version == 0 {
			spec, ok := values["spec"].([]any)
			if !ok || len(spec) != 1 {
				return fmt.Errorf("expected exactly one spec element")
			}
		}
		return podspec.UpgradeResourcesState(values, "state", []string{"spec"}, version == 0)
	})
	if err != nil {
		diags.AddError(summary, err.Error())
		return state, "", "", diags
	}
	state.Raw = value
	return state, namespace, name, diags
}

func podV1ParseID(id string) (string, string, error) {
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		return "", "", err
	}
	if namespace == "" || name == "" {
		return "", "", fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
	}
	return namespace, name, nil
}

func podV1ValidateSourceIdentity(source *tfprotov6.RawState, namespace, name string) diag.Diagnostics {
	var diags diag.Diagnostics
	if source == nil || len(source.JSON) == 0 {
		return diags
	}

	var identityMap map[string]json.RawMessage
	if err := json.Unmarshal(source.JSON, &identityMap); err != nil {
		diags.AddError(podV1MoveStateErrSummary, fmt.Sprintf("Could not decode the source identity: %s", err))
		return diags
	}
	for key := range identityMap {
		switch key {
		case "namespace", "name", "api_version", "kind":
		default:
			diags.AddError(podV1MoveStateErrSummary, fmt.Sprintf("The source identity contains unsupported attribute %q.", key))
			return diags
		}
	}

	var identity sdkv2PodSourceIdentity
	if err := json.Unmarshal(source.JSON, &identity); err != nil {
		diags.AddError(podV1MoveStateErrSummary, fmt.Sprintf("Could not decode the source identity values: %s", err))
		return diags
	}

	switch {
	case identity.Name != nil && *identity.Name == "":
		diags.AddError(podV1MoveStateErrSummary, "The source identity name is empty.")
	case identity.Name != nil && *identity.Name != name:
		diags.AddError(podV1MoveStateErrSummary,
			fmt.Sprintf("The source identity name %q does not match the source state name %q.", *identity.Name, name))
	}
	switch {
	case identity.Namespace != nil && *identity.Namespace == "":
		diags.AddError(podV1MoveStateErrSummary, "The source identity namespace is empty.")
	case identity.Namespace != nil && *identity.Namespace != namespace:
		diags.AddError(podV1MoveStateErrSummary,
			fmt.Sprintf("The source identity namespace %q does not match the source state namespace %q.", *identity.Namespace, namespace))
	}
	if identity.APIVersion != nil && *identity.APIVersion != "" && *identity.APIVersion != podAPIVersion {
		diags.AddError(podV1MoveStateErrSummary,
			fmt.Sprintf("The source identity api_version %q does not match %q.", *identity.APIVersion, podAPIVersion))
	}
	if identity.Kind != nil && *identity.Kind != "" && *identity.Kind != podKind {
		diags.AddError(podV1MoveStateErrSummary,
			fmt.Sprintf("The source identity kind %q does not match %q.", *identity.Kind, podKind))
	}

	return diags
}
