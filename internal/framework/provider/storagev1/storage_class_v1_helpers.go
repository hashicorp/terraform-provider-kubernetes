// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── String map helpers ────────────────────────────────────────────────────────

// expandStringMap converts map[string]types.String → map[string]string for
// Kubernetes API calls (annotations, labels, parameters).
func expandStringMap(m map[string]types.String) map[string]string {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}

// flattenStringMap converts map[string]string → map[string]types.String.
func flattenStringMap(m map[string]string) map[string]types.String {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]types.String, len(m))
	for k, v := range m {
		result[k] = types.StringValue(v)
	}
	return result
}

// toStringInterfaceMap converts map[string]types.String → map[string]interface{}
// as required by kubernetes.DiffStringMap.
func toStringInterfaceMap(m map[string]types.String) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}

// ── Mount options helpers ─────────────────────────────────────────────────────

// expandMountOptions converts a types.Set of strings → []string for the
// Kubernetes StorageClass MountOptions field.
func expandMountOptions(ctx context.Context, s types.Set) []string {
	if s.IsNull() || s.IsUnknown() || len(s.Elements()) == 0 {
		return nil
	}
	var elems []string
	s.ElementsAs(ctx, &elems, false)
	return elems
}

// flattenMountOptions converts []string → types.Set (ElementType: StringType).
func flattenMountOptions(ctx context.Context, opts []string) types.Set {
	if len(opts) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{})
	}
	elems := make([]attr.Value, len(opts))
	for i, o := range opts {
		elems[i] = types.StringValue(o)
	}
	result, _ := types.SetValue(types.StringType, elems)
	return result
}

// ── Allowed topologies helpers ────────────────────────────────────────────────

// expandAllowedTopologies converts []AllowedTopologyModel →
// []corev1.TopologySelectorTerm for the Kubernetes StorageClass API.
func expandAllowedTopologies(ctx context.Context, topologies []AllowedTopologyModel) []corev1.TopologySelectorTerm {
	if len(topologies) == 0 {
		return nil
	}
	terms := make([]corev1.TopologySelectorTerm, 0, len(topologies))
	for _, t := range topologies {
		term := corev1.TopologySelectorTerm{
			MatchLabelExpressions: expandMatchLabelExpressions(ctx, t.MatchLabelExpressions),
		}
		terms = append(terms, term)
	}
	return terms
}

// expandMatchLabelExpressions converts []MatchLabelExpressionModel →
// []corev1.TopologySelectorLabelRequirement.
func expandMatchLabelExpressions(ctx context.Context, exprs []MatchLabelExpressionModel) []corev1.TopologySelectorLabelRequirement {
	if len(exprs) == 0 {
		return nil
	}
	reqs := make([]corev1.TopologySelectorLabelRequirement, 0, len(exprs))
	for _, e := range exprs {
		var values []string
		e.Values.ElementsAs(ctx, &values, false)
		reqs = append(reqs, corev1.TopologySelectorLabelRequirement{
			Key:    e.Key.ValueString(),
			Values: values,
		})
	}
	return reqs
}

// flattenAllowedTopologies converts []corev1.TopologySelectorTerm →
// []AllowedTopologyModel for Terraform state.
func flattenAllowedTopologies(ctx context.Context, terms []corev1.TopologySelectorTerm) []AllowedTopologyModel {
	if len(terms) == 0 {
		return nil
	}
	result := make([]AllowedTopologyModel, 0, len(terms))
	for _, t := range terms {
		result = append(result, AllowedTopologyModel{
			MatchLabelExpressions: flattenMatchLabelExpressions(ctx, t.MatchLabelExpressions),
		})
	}
	return result
}

// flattenMatchLabelExpressions converts []corev1.TopologySelectorLabelRequirement →
// []MatchLabelExpressionModel.
func flattenMatchLabelExpressions(ctx context.Context, reqs []corev1.TopologySelectorLabelRequirement) []MatchLabelExpressionModel {
	if len(reqs) == 0 {
		return nil
	}
	result := make([]MatchLabelExpressionModel, 0, len(reqs))
	for _, r := range reqs {
		elems := make([]attr.Value, len(r.Values))
		for i, v := range r.Values {
			elems[i] = types.StringValue(v)
		}
		valSet, _ := types.SetValue(types.StringType, elems)
		result = append(result, MatchLabelExpressionModel{
			Key:    types.StringValue(r.Key),
			Values: valSet,
		})
	}
	return result
}

// ── Metadata helpers ──────────────────────────────────────────────────────────

// flattenMetadata converts a Kubernetes ObjectMeta to MetadataModel,
// filtering out internal Kubernetes keys and user-configured ignore patterns.
// current holds the existing Terraform-managed metadata (preserves user-managed keys).
func flattenMetadata(meta metav1.ObjectMeta, current MetadataModel, ignoreAnnotations, ignoreLabels []string) MetadataModel {
	result := MetadataModel{
		Name:            types.StringValue(meta.Name),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
	}

	// generate_name: only set if non-empty to avoid diff vs nil.
	if meta.GenerateName != "" {
		result.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		result.GenerateName = types.StringNull()
	}

	filtered := filterIgnoredMetadataKeys(meta.Annotations, current.Annotations, ignoreAnnotations)
	if len(filtered) > 0 {
		result.Annotations = flattenStringMap(filtered)
	}

	filtered = filterIgnoredMetadataKeys(meta.Labels, current.Labels, ignoreLabels)
	if len(filtered) > 0 {
		result.Labels = flattenStringMap(filtered)
	}

	return result
}

// filterIgnoredMetadataKeys removes internal Kubernetes keys and keys matching
// ignore patterns, unless the key is already present in current (managed by TF).
func filterIgnoredMetadataKeys(meta map[string]string, current map[string]types.String, ignorePatterns []string) map[string]string {
	result := make(map[string]string, len(meta))
	for k, v := range meta {
		_, managedByTF := current[k]
		if !managedByTF && (kubernetes.IsInternalKey(k) || kubernetes.IgnoreKey(k, ignorePatterns)) {
			continue
		}
		result[k] = v
	}
	return result
}
