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

// MetadataPatchOps builds the JSON Patch operations for a metadata update, the framework
// counterpart of SDKv2's patchMetadata (kubernetes/structures.go). pathPrefix is the JSON
// Pointer to the metadata object, e.g. "/metadata/". Migrated resources should use this
// rather than calling DiffStringMap directly, so the skip rule stays in one place.
// Emptiness is measured after expansion, so null and {} count alike: they differ in
// Terraform, but neither names a key to change in Kubernetes.
//
// KNOWN LIMITATION — adding the first managed key to a map is still destructive. An empty
// prior map with a populated planned map takes DiffStringMap's whole-object add branch and
// replaces keys other controllers own. This is pre-existing behaviour shared with every
// SDKv2 resource using patchMetadata, so a fix belongs in a provider-wide change: per-key
// add operations (which need the object read first, since they fail with 422 when the
// parent member is absent) or a JSON merge patch for these two maps.
func MetadataPatchOps(pathPrefix string, state, plan MetadataModel) kubernetes.PatchOperations {
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
	var diags diag.Diagnostics

	meta := metav1.ObjectMeta{}
	if len(in) == 0 {
		return meta, diags
	}
	m := in[0]

	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		meta.Name = m.Name.ValueString()
	}
	if !m.GenerateName.IsNull() && !m.GenerateName.IsUnknown() {
		meta.GenerateName = m.GenerateName.ValueString()
	}
	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		labels := make(map[string]string, len(m.Labels.Elements()))
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		meta.Labels = labels
	}
	if !m.Annotations.IsNull() && !m.Annotations.IsUnknown() {
		annotations := make(map[string]string, len(m.Annotations.Elements()))
		diags.Append(m.Annotations.ElementsAs(ctx, &annotations, false)...)
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

// FlattenMetadata converts a Kubernetes ObjectMeta into the Terraform model.
// Annotations and labels are filtered by removing internal and ignored keys. The API server and other
// controllers add keys the practitioner never wrote, and recording them would show a
// permanent diff. A key present in prior state is never removed.
func FlattenMetadata(ctx context.Context, k8MetaObj metav1.ObjectMeta, prior []MetadataModel, ignoreAnnotations, ignoreLabels []string) ([]MetadataModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var newMeta MetadataModel

	var priorMeta MetadataModel
	if len(prior) > 0 {
		priorMeta = prior[0]
	}

	annotations, d := filterMetadataMap(ctx, k8MetaObj.Annotations, priorMeta.Annotations, ignoreAnnotations)
	diags.Append(d...)
	newMeta.Annotations = annotations

	labels, d := filterMetadataMap(ctx, k8MetaObj.Labels, priorMeta.Labels, ignoreLabels)
	diags.Append(d...)
	newMeta.Labels = labels

	// generate_name is Optional but not Computed, so an absent value must stay null
	// rather than becoming "". SDKv2 achieved the same by omitting the map key.
	if k8MetaObj.GenerateName != "" {
		newMeta.GenerateName = types.StringValue(k8MetaObj.GenerateName)
	} else {
		newMeta.GenerateName = types.StringNull()
	}

	newMeta.Generation = types.Int64Value(k8MetaObj.Generation)
	newMeta.Name = types.StringValue(k8MetaObj.Name)
	newMeta.ResourceVersion = types.StringValue(k8MetaObj.ResourceVersion)
	newMeta.UID = types.StringValue(string(k8MetaObj.UID))

	return []MetadataModel{newMeta}, diags
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
