// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// expandMetadata converts the Terraform model into a Kubernetes ObjectMeta.
//
// Null and unknown values are left as the zero value rather than written through,
// so that an omitted attribute produces an absent field rather than an empty one.
func expandMetadata(ctx context.Context, in []MetadataModel) (metav1.ObjectMeta, diag.Diagnostics) {
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

// expandMapForPatch converts a types.Map into the map[string]interface{} shape that
// kubernetes.DiffStringMap expects.
//
// The values matter here, unlike the `declared` set in filterMetadataMap where only
// key presence is consulted: DiffStringMap asserts v.(string) without comma-ok when
// building Add and Replace operations, so a nil value panics, and it compares old
// against new values to decide between add and replace.
//
// A null or unknown map yields an empty map, which DiffStringMap treats as "no prior
// keys" — emitting a single Add for the whole object rather than per-key operations.
func expandMapForPatch(m types.Map) map[string]interface{} {
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

// flattenMetadata converts a Kubernetes ObjectMeta into the Terraform model, applying
// the two filtering rules documented in MIGRATION_FINDINGS_namespace_v1.md §2.
//
// prior is the previous state for this resource, used as the config-membership
// reference: a key the practitioner declared survives both rules. It is a slice to
// mirror expandMetadata and the ListNestedBlock it maps to; an empty slice (the
// import case) yields an all-null baseline, so nothing is exempt from filtering.
func flattenMetadata(ctx context.Context, k8MetaObj metav1.ObjectMeta, prior []MetadataModel, ignoreAnnotations, ignoreLabels []string) ([]MetadataModel, diag.Diagnostics) {
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

// filterMetadataMap applies both removal rules to one metadata map and converts the
// result to a types.Map.
//
// The removals always run, including when prior is null. That is the case that
// matters most: a namespace with no declared labels still comes back from the API
// carrying kubernetes.io/metadata.name, and skipping the filter there would write
// it into state.
func filterMetadataMap(ctx context.Context, fromAPI map[string]string, prior types.Map, ignore []string) (types.Map, diag.Diagnostics) {
	// Keys the practitioner declared. Only presence is consulted (isKeyInMap), so
	// the values are irrelevant. Empty when prior is null or unknown.
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

	// Mirror prior state's null-ness when nothing survives. Go leaves a non-nil,
	// zero-length map after deleting the last key, and types.MapValueFrom would turn
	// that into a known empty map — which differs from null and produces a permanent
	// diff. See MIGRATION_FINDINGS_namespace_v1.md §3.
	if len(filtered) == 0 && prior.IsNull() {
		return types.MapNull(types.StringType), nil
	}

	// MapValueFrom cannot actually fail for map[string]string into StringType, but
	// its diagnostics are propagated rather than discarded so a future change to the
	// element type surfaces as an error instead of a silent wrong value.
	return types.MapValueFrom(ctx, types.StringType, filtered)
}
