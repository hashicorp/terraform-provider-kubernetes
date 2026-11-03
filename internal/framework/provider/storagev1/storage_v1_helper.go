// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

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

// patchAllowedTopologies returns the JSON patch operations that move
// allowedTopologies from the state value to the planned value. "add" both
// creates and replaces the field. Clearing also uses "add" with an empty list
// rather than "remove", because the field may already be absent on the server
// (e.g. removed out of band), and removing a nonexistent key fails the patch.
func patchAllowedTopologies(ctx context.Context, state, plan []AllowedTopologyModel) kubernetes.PatchOperations {
	oldVal := expandAllowedTopologies(ctx, state)
	newVal := expandAllowedTopologies(ctx, plan)
	if reflect.DeepEqual(oldVal, newVal) {
		return nil
	}
	if newVal == nil {
		newVal = []corev1.TopologySelectorTerm{}
	}
	return kubernetes.PatchOperations{&kubernetes.AddOperation{
		Path:  "/allowedTopologies",
		Value: newVal,
	}}
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
