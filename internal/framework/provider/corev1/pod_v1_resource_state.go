// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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

// UpgradeState implements [resource.ResourceWithUpgradeState].
func (p *PodV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				if req.RawState == nil || len(req.RawState.JSON) == 0 {
					resp.Diagnostics.AddError(podV1UpgradeStateErrSummary,
						"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
					return
				}

				stateMap, _, _, diags := podV1DecodeSourceState(req.RawState.JSON, 0, podV1UpgradeStateErrSummary)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(podV1SetStateFromJSONMap(ctx, &resp.State, stateMap, podV1UpgradeStateErrSummary)...)
			},
		},
	}
}

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_pod.example
//	  to   = kubernetes_pod_v1.example
//	}
//
// Do not guard on SourceIdentitySchemaVersion: terraform-plugin-go v0.29.0
// still leaves it at zero for all sources.
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

				if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
					resp.Diagnostics.AddError(podV1MoveStateErrSummary,
						"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
					return
				}

				stateMap, namespace, name, diags := podV1DecodeSourceState(req.SourceRawState.JSON, req.SourceSchemaVersion, podV1MoveStateErrSummary)
				resp.Diagnostics.Append(diags...)
				resp.Diagnostics.Append(podV1ValidateSourceIdentity(req.SourceIdentity, namespace, name)...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(podV1SetStateFromJSONMap(ctx, &resp.TargetState, stateMap, podV1MoveStateErrSummary)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
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

type sdkv2PodState struct {
	ID       string               `json:"id"`
	Metadata []sdkv2PodMetadataV1 `json:"metadata"`
}

type sdkv2PodMetadataV1 struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type sdkv2PodSourceIdentity struct {
	APIVersion *string `json:"api_version"`
	Kind       *string `json:"kind"`
	Name       *string `json:"name"`
	Namespace  *string `json:"namespace"`
}

var podStateTopLevelAttrs = map[string]struct{}{
	"id": {}, "metadata": {}, "spec": {}, "target_state": {}, "timeouts": {},
}

var podStateMetadataAttrs = map[string]struct{}{
	"annotations": {}, "generate_name": {}, "generation": {}, "labels": {},
	"name": {}, "namespace": {}, "resource_version": {}, "uid": {},
}

var podStateTimeoutAttrs = map[string]struct{}{
	"create": {}, "delete": {},
}

func podV1DecodeSourceState(sourceJSON []byte, sourceSchemaVersion int64, summary string) (map[string]any, string, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	var root map[string]json.RawMessage
	if err := json.Unmarshal(sourceJSON, &root); err != nil {
		diags.AddError(summary, fmt.Sprintf("Could not decode the source state JSON object: %s", err))
		return nil, "", "", diags
	}
	for key := range root {
		if _, ok := podStateTopLevelAttrs[key]; !ok {
			diags.AddError(summary, fmt.Sprintf("The source state contains unsupported top-level attribute %q.", key))
			return nil, "", "", diags
		}
	}

	var prior sdkv2PodState
	if err := json.Unmarshal(sourceJSON, &prior); err != nil {
		diags.AddError(summary, fmt.Sprintf("Could not decode the source state: %s", err))
		return nil, "", "", diags
	}
	if prior.ID == "" {
		diags.AddError(summary, "The source state has an empty id.")
		return nil, "", "", diags
	}
	namespace, name, err := podV1ParseID(prior.ID)
	if err != nil {
		diags.AddError(summary, err.Error())
		return nil, "", "", diags
	}
	if len(prior.Metadata) != 1 {
		diags.AddError(summary, fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(prior.Metadata)))
		return nil, "", "", diags
	}
	metadata := prior.Metadata[0]
	if metadata.Namespace != namespace || metadata.Name != name {
		diags.AddError(summary, fmt.Sprintf("The source id %q does not match metadata namespace/name %q/%q.", prior.ID, metadata.Namespace, metadata.Name))
		return nil, "", "", diags
	}

	if rawMetadata, ok := root["metadata"]; ok {
		var metadataList []map[string]json.RawMessage
		if err := json.Unmarshal(rawMetadata, &metadataList); err != nil {
			diags.AddError(summary, fmt.Sprintf("Could not decode metadata in the source state: %s", err))
			return nil, "", "", diags
		}
		if len(metadataList) != 1 {
			diags.AddError(summary, fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(metadataList)))
			return nil, "", "", diags
		}
		for key := range metadataList[0] {
			if _, ok := podStateMetadataAttrs[key]; !ok {
				diags.AddError(summary, fmt.Sprintf("The source state metadata contains unsupported attribute %q.", key))
				return nil, "", "", diags
			}
		}
	}

	var state map[string]any
	stateDecoder := json.NewDecoder(bytes.NewReader(sourceJSON))
	stateDecoder.UseNumber()
	if err := stateDecoder.Decode(&state); err != nil {
		diags.AddError(summary, fmt.Sprintf("Could not decode source state values: %s", err))
		return nil, "", "", diags
	}
	if err := ensureEOF(stateDecoder); err != nil {
		diags.AddError(summary, fmt.Sprintf("Could not decode source state values: %s", err))
		return nil, "", "", diags
	}

	normalizedTimeouts, timeoutDiags := podV1NormalizeSourceTimeouts(root, summary)
	diags.Append(timeoutDiags...)
	if diags.HasError() {
		return nil, "", "", diags
	}
	state["timeouts"] = normalizedTimeouts

	if sourceSchemaVersion == 0 {
		if err := podV1UpgradeResourcesFieldV0(state); err != nil {
			diags.AddError(summary, fmt.Sprintf("Could not upgrade schema version 0 resources fields: %s", err))
			return nil, "", "", diags
		}
	}

	return state, namespace, name, diags
}

func podV1NormalizeSourceTimeouts(root map[string]json.RawMessage, summary string) (any, diag.Diagnostics) {
	var diags diag.Diagnostics

	rawTimeouts, ok := root["timeouts"]
	if !ok || string(rawTimeouts) == "null" {
		return nil, diags
	}

	var timeoutsObject map[string]json.RawMessage
	if err := json.Unmarshal(rawTimeouts, &timeoutsObject); err != nil {
		diags.AddError(summary, fmt.Sprintf("Could not decode source timeouts: %s", err))
		return nil, diags
	}
	for key := range timeoutsObject {
		if _, ok := podStateTimeoutAttrs[key]; !ok {
			diags.AddError(summary, fmt.Sprintf("Source timeouts contains unsupported attribute %q.", key))
			return nil, diags
		}
	}

	normalized := map[string]any{"create": nil, "delete": nil}
	for _, key := range []string{"create", "delete"} {
		rawValue, exists := timeoutsObject[key]
		if !exists || string(rawValue) == "null" {
			continue
		}
		var value string
		if err := json.Unmarshal(rawValue, &value); err != nil {
			diags.AddError(summary, fmt.Sprintf("Timeout %q in source state is not a string or null: %s", key, err))
			return nil, diags
		}
		if value != "" {
			normalized[key] = value
		}
	}

	return normalized, diags
}

func podV1UpgradeResourcesFieldV0(state map[string]any) error {
	specRaw, ok := state["spec"]
	if !ok || specRaw == nil {
		return fmt.Errorf("spec is missing")
	}

	specList, ok := specRaw.([]any)
	if !ok {
		return fmt.Errorf("spec is %T, expected list", specRaw)
	}
	if len(specList) != 1 {
		return fmt.Errorf("spec has %d elements, expected exactly 1", len(specList))
	}

	spec, ok := specList[0].(map[string]any)
	if !ok {
		return fmt.Errorf("spec[0] is %T, expected object", specList[0])
	}
	if err := podV1UpgradeContainerResourcesFieldV0(spec, "container", "spec[0].container"); err != nil {
		return err
	}
	if err := podV1UpgradeContainerResourcesFieldV0(spec, "init_container", "spec[0].init_container"); err != nil {
		return err
	}
	return nil
}

func podV1UpgradeContainerResourcesFieldV0(spec map[string]any, fieldName, path string) error {
	containersRaw, ok := spec[fieldName]
	if !ok || containersRaw == nil {
		return nil
	}
	containers, ok := containersRaw.([]any)
	if !ok {
		return fmt.Errorf("%s is %T, expected list", path, containersRaw)
	}

	for i, containerRaw := range containers {
		container, ok := containerRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s[%d] is %T, expected object", path, i, containerRaw)
		}
		resourcesRaw, ok := container["resources"]
		if !ok || resourcesRaw == nil {
			continue
		}
		resourcesList, ok := resourcesRaw.([]any)
		if !ok {
			if _, alreadyMap := resourcesRaw.(map[string]any); alreadyMap {
				continue
			}
			return fmt.Errorf("%s[%d].resources is %T, expected list", path, i, resourcesRaw)
		}
		if len(resourcesList) == 0 {
			continue
		}
		if len(resourcesList) > 1 {
			return fmt.Errorf("%s[%d].resources has %d elements, expected at most 1", path, i, len(resourcesList))
		}
		resources, ok := resourcesList[0].(map[string]any)
		if !ok {
			return fmt.Errorf("%s[%d].resources[0] is %T, expected object", path, i, resourcesList[0])
		}
		if err := podV1UpgradeResourceQuantityFieldV0(resources, "requests", fmt.Sprintf("%s[%d].resources[0].requests", path, i)); err != nil {
			return err
		}
		if err := podV1UpgradeResourceQuantityFieldV0(resources, "limits", fmt.Sprintf("%s[%d].resources[0].limits", path, i)); err != nil {
			return err
		}
	}

	return nil
}

func podV1UpgradeResourceQuantityFieldV0(resources map[string]any, fieldName, path string) error {
	raw, ok := resources[fieldName]
	if !ok || raw == nil {
		resources[fieldName] = map[string]any{}
		return nil
	}

	switch v := raw.(type) {
	case map[string]any:
		return nil
	case []any:
		if len(v) == 0 {
			resources[fieldName] = map[string]any{}
			return nil
		}
		if len(v) > 1 {
			return fmt.Errorf("%s has %d elements, expected at most 1", path, len(v))
		}
		objectValue, ok := v[0].(map[string]any)
		if !ok {
			return fmt.Errorf("%s[0] is %T, expected object", path, v[0])
		}
		resources[fieldName] = objectValue
		return nil
	default:
		return fmt.Errorf("%s is %T, expected list", path, raw)
	}
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

func ensureEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing data after JSON object")
		}
		return err
	}
	return nil
}

func podV1SetStateFromJSONMap(ctx context.Context, state *tfsdk.State, values map[string]any, summary string) diag.Diagnostics {
	var diags diag.Diagnostics

	data, err := json.Marshal(values)
	if err != nil {
		diags.AddError(summary, fmt.Sprintf("Could not encode converted state JSON: %s", err))
		return diags
	}
	raw := tfprotov6.RawState{JSON: data}

	typedValue, err := raw.Unmarshal(state.Schema.Type().TerraformType(ctx))
	if err != nil {
		diags.AddError(summary, fmt.Sprintf("Converted state does not match the current schema: %s", err))
		return diags
	}

	state.Raw = typedValue
	return diags
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
