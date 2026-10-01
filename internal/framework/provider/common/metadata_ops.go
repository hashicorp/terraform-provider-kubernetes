// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
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

// FlattenDataSourceMetadataFields converts a Kubernetes ObjectMeta into the data source metadata
// model. It is the counterpart of SDKv2's flattenMetadataFields (kubernetes/structures.go),
// and the important thing about it is what it does *not* do: no filtering.
//
// The resource path strips internal keys and anything matching ignore_labels /
// ignore_annotations, because a resource has to distinguish what the practitioner manages
// from what the cluster added. A data source manages nothing — it reports what is there — so
// SDKv2 passes the API response through untouched, and keys like
// kubernetes.io/metadata.name are expected to appear. Filtering here would hide data that
// practitioners read today.
func FlattenDataSourceMetadataFields(ctx context.Context, k8MetaObj metav1.ObjectMeta) (MetadataBase, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out MetadataBase

	// Empty, not null — matching what SDKv2 leaves in a data source's state.
	//
	// SDKv2 is not consistent between the two paths, and the reason is normalizeNullValues
	// in helper/schema/grpc_provider.go: it converts an empty collection back to null, and
	// it runs for ReadResource, PlanResourceChange and ApplyResourceChange but not for
	// ReadDataSource. So an object with no annotations ends up null in a resource's state
	// and {} in a data source's. MapValueFrom on a nil Go map produces null, so without
	// this normalisation the migrated data source would report a diff on upgrade.
	annotations, d := stringMapValue(ctx, k8MetaObj.Annotations)
	diags.Append(d...)
	out.Annotations = annotations

	labels, d := stringMapValue(ctx, k8MetaObj.Labels)
	diags.Append(d...)
	out.Labels = labels

	out.Generation = types.Int64Value(k8MetaObj.Generation)
	out.Name = types.StringValue(k8MetaObj.Name)
	out.ResourceVersion = types.StringValue(k8MetaObj.ResourceVersion)
	out.UID = types.StringValue(string(k8MetaObj.UID))

	return out, diags
}

// FlattenNamespacedMetadataFields is FlattenMetadataFields for a namespaced data source.
func FlattenNamespacedMetadataFields(ctx context.Context, k8MetaObj metav1.ObjectMeta) (NamespacedMetadataBase, diag.Diagnostics) {
	base, diags := FlattenDataSourceMetadataFields(ctx, k8MetaObj)
	return NamespacedMetadataBase{
		MetadataBase: base,
		Namespace:    types.StringValue(k8MetaObj.Namespace),
	}, diags
}

// stringMapValue converts a Kubernetes metadata map to types.Map, treating nil as empty
// rather than null. See FlattenMetadataFields for why.
func stringMapValue(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		m = map[string]string{}
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

// NormalizeNotFoundMetadata gives a data source's metadata the state SDKv2 recorded when
// the object does not exist: fields the configuration set are kept, and each unset field
// becomes a zero value — {} for annotations and labels, "" for uid and resource_version,
// 0 for generation — rather than null. Null values inside configured annotations or labels
// are dropped, as SDKv2 dropped them.
//
// Preserving this is deliberate. The read does not error on a 404, so these values are how
// practitioners detect absence, and the common idiom is metadata[0].uid != "". Returning
// null would make that comparison true for a missing object, because HCL treats null as
// equal only to null, silently inverting existence checks on upgrade.
//
// Attributes populated only from the API response, such as a namespace's spec, are a
// different case: SDKv2 never set them on a 404, so they stay null and callers leave them
// alone. Namespaced callers normalize the embedded MetadataBase and keep their namespace.
func NormalizeNotFoundMetadata(in MetadataBase) MetadataBase {
	out := in
	out.Annotations = notFoundStringMap(out.Annotations)
	out.Labels = notFoundStringMap(out.Labels)
	if out.Generation.IsNull() {
		out.Generation = types.Int64Value(0)
	}
	if out.ResourceVersion.IsNull() {
		out.ResourceVersion = types.StringValue("")
	}
	if out.UID.IsNull() {
		out.UID = types.StringValue("")
	}
	return out
}

// notFoundStringMap is NormalizeNotFoundMetadata's rule for annotations and labels: null
// becomes {}, and a configured map keeps its entries except null ones.
func notFoundStringMap(m types.Map) types.Map {
	if m.IsNull() {
		return types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	if m.IsUnknown() {
		return m
	}
	elems := m.Elements()
	kept := make(map[string]attr.Value, len(elems))
	for k, v := range elems {
		if !v.IsNull() {
			kept[k] = v
		}
	}
	if len(kept) == len(elems) {
		return m
	}
	return types.MapValueMust(types.StringType, kept)
}

// ResolveDataSourceNamespace returns the namespace a namespaced data source should read,
// applying SDKv2's "default" fallback when the configuration omits one.
//
// SDKv2 gets this from a schema default. The framework cannot: datasource/schema
// attributes have no Default field, because defaults are applied during plan
// modification and a data source has no plan. So the fallback has to happen in Read.
//
// Callers write the result back into the model as well as using it for the API call.
// SDKv2's default lands in state, so a configuration that omits namespace records
// "default"; returning null instead would change state shape.
func ResolveDataSourceNamespace(in types.String) types.String {
	if in.IsNull() || in.IsUnknown() || in.ValueString() == "" {
		return types.StringValue(corev1.NamespaceDefault)
	}
	return in
}
