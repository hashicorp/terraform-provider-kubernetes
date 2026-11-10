// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*ConfigMapV1)(nil)
	_ resource.ResourceWithConfigure       = (*ConfigMapV1)(nil)
	_ resource.ResourceWithIdentity        = (*ConfigMapV1)(nil)
	_ resource.ResourceWithImportState     = (*ConfigMapV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*ConfigMapV1)(nil)
	_ resource.ResourceWithMoveState       = (*ConfigMapV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ConfigMapV1)(nil)
)

const (
	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"

	deprecatedConfigMapTypeName      = "kubernetes_config_map"
	deprecatedConfigMapSchemaVersion = 0

	configMapMoveStateErrSummary = "Unable to move kubernetes_config_map state"
)

type ConfigMapV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in ProviderData by
	// internal/framework/provider/provider_configure.go. Assert the *result* — see sdkv2Meta().
	SDKv2Meta func() any
}

func NewConfigMapV1() resource.Resource {
	return &ConfigMapV1{}
}

// Metadata implements [resource.Resource].
func (c *ConfigMapV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_map_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (c *ConfigMapV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	c.SDKv2Meta = sdkv2Meta
}

func (c *ConfigMapV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	return common.ResolveSDKv2Meta(c.SDKv2Meta)
}

// ModifyPlan implements [resource.ResourceWithModifyPlan]. It ports the SDKv2
// CustomizeDiff: once the object is immutable in state, any change to data, binary_data or
// immutable requires replacement, because the API server rejects such updates. Metadata
// stays mutable.
//
// The test is on the *prior* value of immutable, as in SDKv2 (`diff.GetChange` old side), so
// turning immutable on is an ordinary update while changing anything afterwards replaces.
// Null and {} compare equal: SDKv2 state cannot distinguish them, and replacing an
// immutable ConfigMap because of that would destroy data the user did not touch.
func (c *ConfigMapV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Create or destroy: nothing to compare.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan ConfigMapV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.Immutable.ValueBool() {
		return
	}

	if !mapsEquivalent(state.Data, plan.Data) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("data"))
	}
	if !mapsEquivalent(state.BinaryData, plan.BinaryData) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("binary_data"))
	}
	if !plan.Immutable.IsUnknown() && plan.Immutable.ValueBool() != state.Immutable.ValueBool() {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("immutable"))
	}
}

// ImportState implements [resource.ResourceWithImportState]. It accepts "namespace/name" or
// a resource identity (namespace defaults to "default" when omitted). Read validates the ID
// and populates state and identity.
func (c *ConfigMapV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" {
		var identityData common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identityData.Kind.ValueString() != configMapKind || identityData.APIVersion.ValueString() != configMapAPIVersion {
			resp.Diagnostics.AddError("invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					configMapKind, configMapAPIVersion, identityData.Kind.ValueString(), identityData.APIVersion.ValueString()))
			return
		}
		namespace := identityData.Namespace.ValueString()
		if namespace == "" {
			namespace = metav1.NamespaceDefault
		}
		id = kubernetes.BuildId(metav1.ObjectMeta{Namespace: namespace, Name: identityData.Name.ValueString()})
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// Without this, any ConfigMap created by provider 2.37.x or older fails its first plan with
// "Unable to Upgrade Resource Identity": identity shipped in 2.38.0, so older state carries
// identity_schema_version 0 and no identity. SDKv2 answers that generically; the framework
// requires each resource to supply it.
func (c *ConfigMapV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(configMapKind, configMapAPIVersion)
}

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_config_map.example
//	  to   = kubernetes_config_map_v1.example
//	}
//
// the supported route off the deprecated alias, which stays on SDKv2. The source is decoded
// from raw JSON into a frozen struct rather than through a SourceSchema, so it does not
// follow this resource's schema as that changes.
func (c *ConfigMapV1) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{{StateMover: c.moveFromDeprecatedConfigMap}}
}

func (c *ConfigMapV1) moveFromDeprecatedConfigMap(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	// Do not guard on req.SourceIdentitySchemaVersion: terraform-plugin-go never copies it
	// off the wire, so it is always 0.
	if req.SourceTypeName != deprecatedConfigMapTypeName ||
		req.SourceSchemaVersion != deprecatedConfigMapSchemaVersion ||
		!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
		// Skipping returns no state and no diagnostics; leave a reason in the log.
		tflog.Debug(ctx, "MoveState: not a kubernetes_config_map v0 source, skipping", map[string]any{
			"source_type_name":        req.SourceTypeName,
			"source_schema_version":   req.SourceSchemaVersion,
			"source_provider_address": req.SourceProviderAddress,
		})
		return
	}
	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError(configMapMoveStateErrSummary,
			"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
		return
	}

	var src sdkv2ConfigMapStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &src); err != nil {
		resp.Diagnostics.AddError(configMapMoveStateErrSummary, fmt.Sprintf("Could not decode the source state: %s", err))
		return
	}
	if src.ID == "" {
		resp.Diagnostics.AddError(configMapMoveStateErrSummary, "The source state has an empty id.")
		return
	}
	if len(src.Metadata) != 1 {
		resp.Diagnostics.AddError(configMapMoveStateErrSummary,
			fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(src.Metadata)))
		return
	}
	m := src.Metadata[0]

	annotations, d := common.SDKv2MapToFramework(ctx, m.Annotations)
	resp.Diagnostics.Append(d...)
	labels, d := common.SDKv2MapToFramework(ctx, m.Labels)
	resp.Diagnostics.Append(d...)
	data, d := common.SDKv2MapToFramework(ctx, src.Data)
	resp.Diagnostics.Append(d...)
	binaryData, d := common.SDKv2MapToFramework(ctx, src.BinaryData)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// SDKv2 writes "" for an unset generate_name, where the framework needs null. Left as
	// "", a plan with -refresh=false compares "" against null and recreates the object.
	generateName := types.StringNull()
	if m.GenerateName != "" {
		generateName = types.StringValue(m.GenerateName)
	}

	target := ConfigMapV1Model{
		ID: types.StringValue(src.ID),
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations:     annotations,
					Generation:      types.Int64Value(m.Generation),
					Labels:          labels,
					Name:            types.StringValue(m.Name),
					ResourceVersion: types.StringValue(m.ResourceVersion),
					UID:             types.StringValue(m.UID),
				},
				GenerateName: generateName,
			},
			Namespace: types.StringValue(m.Namespace),
		}},
		BinaryData: binaryData,
		Data:       data,
		Immutable:  types.BoolValue(src.Immutable),
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
	if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
		return
	}
	resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, configMapIdentity(m.Namespace, m.Name))...)
}

// expandStringMap converts a types.Map of strings into a Go map. Null and unknown give nil.
func expandStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// expandBase64Map is expandBase64MapToByteMap from kubernetes/structures.go. The schema
// validator rejects invalid base64 first, so a failure here is reported rather than,
// as in SDKv2, silently dropping the entry.
func expandBase64Map(ctx context.Context, m types.Map) (map[string][]byte, diag.Diagnostics) {
	strs, diags := expandStringMap(ctx, m)
	if diags.HasError() || strs == nil {
		return nil, diags
	}
	out := make(map[string][]byte, len(strs))
	for k, v := range strs {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			diags.AddError("Invalid base64 in binary_data", fmt.Sprintf("Key %q is not base64-encoded: %s", k, err))
			continue
		}
		out[k] = b
	}
	return out, diags
}

// flattenStringMap converts an API map into state. An empty API map stays null when the
// prior value was null, because data and binary_data are Optional without Computed: a
// known {} there would be inconsistent with a null plan. A prior known {} is preserved.
func flattenStringMap(ctx context.Context, in map[string]string, prior types.Map) (types.Map, diag.Diagnostics) {
	if len(in) == 0 && prior.IsNull() {
		return types.MapNull(types.StringType), nil
	}
	if in == nil {
		in = map[string]string{}
	}
	return types.MapValueFrom(ctx, types.StringType, in)
}

// flattenBase64Map is flattenByteMapToBase64Map from kubernetes/structures.go with the
// null/empty handling of flattenStringMap.
func flattenBase64Map(ctx context.Context, in map[string][]byte, prior types.Map) (types.Map, diag.Diagnostics) {
	encoded := make(map[string]string, len(in))
	for k, v := range in {
		encoded[k] = base64.StdEncoding.EncodeToString(v)
	}
	return flattenStringMap(ctx, encoded, prior)
}

// mapsEquivalent reports whether two maps carry the same content, treating null and an
// empty map as equal. SDKv2 could not tell them apart, so state written by it holds null
// where the configuration may say {}. Treating that as a change would patch needlessly and,
// for an immutable ConfigMap, would plan a replacement.
func mapsEquivalent(a, b types.Map) bool {
	if a.IsUnknown() || b.IsUnknown() {
		return a.IsUnknown() && b.IsUnknown()
	}
	if len(a.Elements()) == 0 && len(b.Elements()) == 0 {
		return true
	}
	return a.Equal(b)
}

// parseConfigMapID parses "namespace/name". kubernetes.IdParts only checks the separator,
// so empty components are rejected here as well.
func parseConfigMapID(id string) (namespace, name string, err error) {
	namespace, name, ok := strings.Cut(id, "/")
	if !ok || namespace == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("Unexpected ID format (%q), expected %q with a non-empty namespace and name.", id, "namespace/name")
	}
	return namespace, name, nil
}

func configMapIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(configMapAPIVersion),
			Kind:       types.StringValue(configMapKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}
