// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetadataPatchOps mirrors SDKv2 patchMetadata, skipping maps with no managed keys.
// pathPrefix is the metadata JSON Pointer, e.g. "/metadata/".
// Adding the first managed key still replaces the whole map, as in SDKv2;
// see FIX_namespace_v1_metadata_patch_ops.md for this provider-wide limitation.
func MetadataPatchOps(pathPrefix string, state, plan MetadataModel) kubernetes.PatchOperations {
	return BaseMetadataPatchOps(pathPrefix, state.MetadataBase, plan.MetadataBase)
}

// BaseMetadataPatchOps diffs the mutable maps shared by all metadata variants.
func BaseMetadataPatchOps(pathPrefix string, state, plan MetadataBase) kubernetes.PatchOperations {
	ops := make(kubernetes.PatchOperations, 0)

	oldAnnotations, newAnnotations := ExpandMapForPatch(state.Annotations), ExpandMapForPatch(plan.Annotations)
	if len(oldAnnotations) > 0 || len(newAnnotations) > 0 {
		ops = append(ops, kubernetes.DiffStringMap(pathPrefix+"annotations", oldAnnotations, newAnnotations)...)
	}

	oldLabels, newLabels := ExpandMapForPatch(state.Labels), ExpandMapForPatch(plan.Labels)
	if len(oldLabels) > 0 || len(newLabels) > 0 {
		ops = append(ops, kubernetes.DiffStringMap(pathPrefix+"labels", oldLabels, newLabels)...)
	}

	return ops
}

// ExpandMetadata converts the Terraform model into a Kubernetes ObjectMeta.
func ExpandMetadata(ctx context.Context, in []MetadataModel) (metav1.ObjectMeta, diag.Diagnostics) {
	if len(in) == 0 {
		return metav1.ObjectMeta{}, nil
	}

	meta, diags := ExpandBaseMetadata(ctx, in[0].MetadataBase)
	if !in[0].GenerateName.IsNull() && !in[0].GenerateName.IsUnknown() {
		meta.GenerateName = in[0].GenerateName.ValueString()
	}
	return meta, diags
}

// ExpandBaseMetadata converts shared metadata fields into a Kubernetes ObjectMeta.
func ExpandBaseMetadata(ctx context.Context, metadata MetadataBase) (metav1.ObjectMeta, diag.Diagnostics) {
	var diags diag.Diagnostics
	meta := metav1.ObjectMeta{}
	if !metadata.Name.IsNull() && !metadata.Name.IsUnknown() {
		meta.Name = metadata.Name.ValueString()
	}

	if !metadata.Labels.IsNull() && !metadata.Labels.IsUnknown() {
		labels := make(map[string]string, len(metadata.Labels.Elements()))
		diags.Append(metadata.Labels.ElementsAs(ctx, &labels, false)...)
		meta.Labels = labels
	}
	if !metadata.Annotations.IsNull() && !metadata.Annotations.IsUnknown() {
		annotations := make(map[string]string, len(metadata.Annotations.Elements()))
		diags.Append(metadata.Annotations.ElementsAs(ctx, &annotations, false)...)
		meta.Annotations = annotations
	}

	return meta, diags
}

// ExpandMapForPatch converts a types.Map into the map[string]interface{} shape that
// kubernetes.DiffStringMap expects.
// A null or unknown map yields an empty map, which DiffStringMap treats as "no prior
// keys" — emitting a single Add for the whole object rather than per-key operations.
func ExpandMapForPatch(m types.Map) map[string]interface{} {
	if m.IsNull() || m.IsUnknown() {
		return map[string]interface{}{}
	}

	// Elements() allocates a defensive copy on every call, so take it once.
	elements := m.Elements()
	out := make(map[string]interface{}, len(elements))
	for k, v := range elements {
		s, ok := v.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		out[k] = s.ValueString()
	}
	return out
}

// FlattenMetadata converts API metadata for Read, filtering internal and ignored keys
// unless previously managed. Keys absent from the API are not restored from prior state.
func FlattenMetadata(ctx context.Context, k8MetaObj metav1.ObjectMeta, prior []MetadataModel, ignoreAnnotations, ignoreLabels []string) ([]MetadataModel, diag.Diagnostics) {
	var priorMeta MetadataBase
	if len(prior) > 0 {
		priorMeta = prior[0].MetadataBase
	}

	base, diags := FlattenBaseMetadata(ctx, k8MetaObj, priorMeta, ignoreAnnotations, ignoreLabels)
	metadata := MetadataModel{MetadataBase: base, GenerateName: types.StringNull()}

	// generate_name is Optional but not Computed, so an absent value must stay null
	// rather than becoming "". SDKv2 achieved the same by omitting the map key.
	if k8MetaObj.GenerateName != "" {
		metadata.GenerateName = types.StringValue(k8MetaObj.GenerateName)
	}

	return []MetadataModel{metadata}, diags
}

// FlattenBaseMetadata converts shared API fields, filtering maps against prior metadata.
func FlattenBaseMetadata(ctx context.Context, k8MetaObj metav1.ObjectMeta, prior MetadataBase, ignoreAnnotations, ignoreLabels []string) (MetadataBase, diag.Diagnostics) {
	annotations, diags := filterMetadataMap(ctx, k8MetaObj.Annotations, prior.Annotations, ignoreAnnotations)
	labels, labelDiags := filterMetadataMap(ctx, k8MetaObj.Labels, prior.Labels, ignoreLabels)
	diags.Append(labelDiags...)

	return MetadataBase{
		Annotations:     annotations,
		Generation:      types.Int64Value(k8MetaObj.Generation),
		Labels:          labels,
		Name:            types.StringValue(k8MetaObj.Name),
		ResourceVersion: types.StringValue(k8MetaObj.ResourceVersion),
		UID:             types.StringValue(string(k8MetaObj.UID)),
	}, diags
}

func filterMetadataMap(ctx context.Context, fromAPI map[string]string, prior types.Map, ignore []string) (types.Map, diag.Diagnostics) {
	declared := map[string]interface{}{}
	if !prior.IsNull() && !prior.IsUnknown() {
		for k := range prior.Elements() {
			declared[k] = nil
		}
	}

	// Copy before mutating: RemoveInternalKeys and RemoveKeys delete in place, and
	// the caller's ObjectMeta should not be modified.
	filtered := make(map[string]string, len(fromAPI))
	for k, v := range fromAPI {
		filtered[k] = v
	}

	kubernetes.RemoveInternalKeys(filtered, declared)
	kubernetes.RemoveKeys(filtered, declared, ignore)

	if len(filtered) == 0 && prior.IsNull() {
		return types.MapNull(types.StringType), nil
	}

	return types.MapValueFrom(ctx, types.StringType, filtered)
}

// ExpandNamespacedMetadata adds namespace to ExpandMetadata's result.
func ExpandNamespacedMetadata(ctx context.Context, in []NamespacedMetadataModel) (metav1.ObjectMeta, diag.Diagnostics) {
	if len(in) == 0 {
		return metav1.ObjectMeta{}, nil
	}

	meta, diags := ExpandMetadata(ctx, []MetadataModel{in[0].MetadataModel})
	if !in[0].Namespace.IsNull() && !in[0].Namespace.IsUnknown() {
		meta.Namespace = in[0].Namespace.ValueString()
	}

	return meta, diags
}

// FlattenNamespacedMetadata adds namespace to FlattenMetadata's result.
func FlattenNamespacedMetadata(ctx context.Context, k8MetaObj metav1.ObjectMeta, prior []NamespacedMetadataModel, ignoreAnnotations, ignoreLabels []string) ([]NamespacedMetadataModel, diag.Diagnostics) {
	var priorBase []MetadataModel
	if len(prior) > 0 {
		priorBase = []MetadataModel{prior[0].MetadataModel}
	}

	base, diags := FlattenMetadata(ctx, k8MetaObj, priorBase, ignoreAnnotations, ignoreLabels)
	return []NamespacedMetadataModel{{MetadataModel: base[0], Namespace: types.StringValue(k8MetaObj.Namespace)}}, diags
}
