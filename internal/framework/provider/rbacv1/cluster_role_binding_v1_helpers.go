// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// expandStringMap converts a Terraform types.Map into a map[string]string.
// A null/unknown map yields a nil map (so it is omitted from the API object).
func expandStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	result := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &result, false)
	if len(result) == 0 {
		return nil, diags
	}
	return result, diags
}

// flattenStringMap converts a map[string]string into a Terraform types.Map.
// An empty/nil map is represented as a null map to match the SDKv2 behaviour
// of omitting empty metadata maps from state.
func flattenStringMap(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elements := make(map[string]interface{}, len(m))
	for k, v := range m {
		elements[k] = v
	}
	result, _ := types.MapValueFrom(context.Background(), types.StringType, elements)
	return result
}

// typesMapElements extracts a map[string]types.String from a types.Map,
// propagating any conversion diagnostics to the caller.
// A null or unknown map returns nil without error.
func typesMapElements(ctx context.Context, value types.Map) (map[string]types.String, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result map[string]types.String
	diags := value.ElementsAs(ctx, &result, false)
	return result, diags
}

// matchesIgnorePattern reports whether key matches any of the regex patterns.
func matchesIgnorePattern(key string, patterns []string) bool {
	for _, pattern := range patterns {
		if ok, _ := regexp.MatchString(pattern, key); ok {
			return true
		}
	}
	return false
}

// isInternalMetadataKey reports whether a Kubernetes metadata key is managed
// internally by the control plane and should not be tracked in Terraform state
// unless the user has explicitly configured it. This mirrors the SDKv2
// isInternalKey helper in kubernetes/structures.go.
func isInternalMetadataKey(key string) bool {
	u, err := url.Parse("//" + key)
	if err != nil {
		return false
	}

	// These application-managed keys are intentionally allowed.
	if u.Hostname() == "app.kubernetes.io" {
		return false
	}

	// AWS load balancer configuration annotations are intentionally allowed.
	if u.Hostname() == "service.beta.kubernetes.io" {
		return false
	}

	if strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}

	if strings.Contains(key, "deprecated.daemonset.template.generation") {
		return true
	}

	return false
}

// filterManagedMetadataKeys returns a filtered copy of m that omits any key
// that is either an internal Kubernetes key or matches an ignore pattern,
// unless that key is already tracked in the current Terraform state (current).
//
// This mirrors the SDKv2 removeInternalKeys + removeKeys behaviour so that
// provider-level ignore_annotations / ignore_labels settings and internal
// Kubernetes keys are handled identically to the original SDKv2 resource.
func filterManagedMetadataKeys(
	m map[string]string,
	current map[string]types.String,
	ignorePatterns []string,
) map[string]string {
	if len(m) == 0 {
		return nil
	}

	result := make(map[string]string, len(m))

	for key, value := range m {
		_, alreadyManaged := current[key]

		if !alreadyManaged &&
			(isInternalMetadataKey(key) || matchesIgnorePattern(key, ignorePatterns)) {
			continue
		}

		result[key] = value
	}

	return result
}

// escapeJSONPointer escapes a string per RFC 6901 so it can be used as a path
// segment in a JSON Patch operation. ~ must be escaped before / to avoid
// double-encoding.
func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	value = strings.ReplaceAll(value, "/", "~1")
	return value
}

// diffStringMap computes the JSON Patch operations needed to transition the
// metadata map at pathPrefix from oldValues to newValues. Only keys present in
// oldValues or newValues are touched; absent keys (externally managed or
// provider-ignored) are never removed.
func diffStringMap(pathPrefix string, oldValues, newValues map[string]string) kubernetes.PatchOperations {
	ops := make(kubernetes.PatchOperations, 0)
	pathPrefix = strings.TrimRight(pathPrefix, "/")

	// When the old map was empty/nil but we now have values, add the whole
	// map in one operation — this matches the SDKv2 diffStringMap behaviour.
	if len(oldValues) == 0 && len(newValues) > 0 {
		v := make(map[string]interface{}, len(newValues))
		for k, val := range newValues {
			v[k] = val
		}
		ops = append(ops, &kubernetes.AddOperation{Path: pathPrefix, Value: v})
		return ops
	}

	// Remove keys that were Terraform-managed (in prior state) but are gone
	// from the plan.
	for k := range oldValues {
		if _, ok := newValues[k]; ok {
			continue
		}
		ops = append(ops, &kubernetes.RemoveOperation{
			Path: pathPrefix + "/" + escapeJSONPointer(k),
		})
	}

	// Add new keys and replace changed values.
	for k, v := range newValues {
		if old, ok := oldValues[k]; ok {
			if old == v {
				continue
			}
			ops = append(ops, &kubernetes.ReplaceOperation{
				Path:  pathPrefix + "/" + escapeJSONPointer(k),
				Value: v,
			})
			continue
		}
		ops = append(ops, &kubernetes.AddOperation{
			Path:  pathPrefix + "/" + escapeJSONPointer(k),
			Value: v,
		})
	}

	return ops
}

// subjectsEqual reports whether two SubjectModel slices are identical in every
// field. The comparison is order-sensitive because subject is a list type.
func subjectsEqual(a, b []SubjectModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Kind.Equal(b[i].Kind) ||
			!a[i].Name.Equal(b[i].Name) ||
			!a[i].APIGroup.Equal(b[i].APIGroup) ||
			!a[i].Namespace.Equal(b[i].Namespace) {
			return false
		}
	}
	return true
}

// buildMetadataPatch computes the JSON Patch operations needed to reconcile
// the metadata annotations and labels from prior state to the new plan.
// Keys not present in either state are not touched, so externally managed
// and provider-ignored metadata is preserved.
func buildMetadataPatch(
	ctx context.Context,
	state ClusterRoleBindingMetadataModel,
	plan ClusterRoleBindingMetadataModel,
) (kubernetes.PatchOperations, diag.Diagnostics) {
	var ops kubernetes.PatchOperations
	var diags diag.Diagnostics

	stateAnnotations, d := expandStringMap(ctx, state.Annotations)
	diags.Append(d...)

	planAnnotations, d := expandStringMap(ctx, plan.Annotations)
	diags.Append(d...)

	stateLabels, d := expandStringMap(ctx, state.Labels)
	diags.Append(d...)

	planLabels, d := expandStringMap(ctx, plan.Labels)
	diags.Append(d...)

	if diags.HasError() {
		return nil, diags
	}

	ops = append(ops, diffStringMap("/metadata/annotations", stateAnnotations, planAnnotations)...)
	ops = append(ops, diffStringMap("/metadata/labels", stateLabels, planLabels)...)

	return ops, diags
}

// expandMetadata builds a Kubernetes ObjectMeta from the (single) metadata block.
func expandMetadata(ctx context.Context, in []ClusterRoleBindingMetadataModel) (metav1.ObjectMeta, diag.Diagnostics) {
	var diags diag.Diagnostics
	meta := metav1.ObjectMeta{}
	if len(in) == 0 {
		return meta, diags
	}
	m := in[0]

	annotations, d := expandStringMap(ctx, m.Annotations)
	diags.Append(d...)
	meta.Annotations = annotations

	labels, d := expandStringMap(ctx, m.Labels)
	diags.Append(d...)
	meta.Labels = labels

	if !m.GenerateName.IsNull() && !m.GenerateName.IsUnknown() {
		meta.GenerateName = m.GenerateName.ValueString()
	}
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		meta.Name = m.Name.ValueString()
	}

	return meta, diags
}

// flattenMetadata produces the metadata block from a Kubernetes ObjectMeta,
// filtering out internal and provider-ignored keys that are not already tracked
// in the current Terraform state (current).
func flattenMetadata(
	ctx context.Context,
	meta metav1.ObjectMeta,
	current ClusterRoleBindingMetadataModel,
	ignoreAnnotations []string,
	ignoreLabels []string,
) ([]ClusterRoleBindingMetadataModel, diag.Diagnostics) {
	currentAnnotations, diags := typesMapElements(ctx, current.Annotations)
	if diags.HasError() {
		return nil, diags
	}

	currentLabels, d := typesMapElements(ctx, current.Labels)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	annotations := filterManagedMetadataKeys(meta.Annotations, currentAnnotations, ignoreAnnotations)
	labels := filterManagedMetadataKeys(meta.Labels, currentLabels, ignoreLabels)

	model := ClusterRoleBindingMetadataModel{
		Annotations:     flattenStringMap(annotations),
		Labels:          flattenStringMap(labels),
		Name:            types.StringValue(meta.Name),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
	}
	if meta.GenerateName != "" {
		model.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		model.GenerateName = types.StringNull()
	}

	return []ClusterRoleBindingMetadataModel{model}, diags
}

// expandRoleRef builds a Kubernetes RoleRef from the (single) role_ref block.
func expandRoleRef(in []RoleRefModel) api.RoleRef {
	if len(in) == 0 {
		return api.RoleRef{}
	}
	r := in[0]
	return api.RoleRef{
		APIGroup: r.APIGroup.ValueString(),
		Kind:     r.Kind.ValueString(),
		Name:     r.Name.ValueString(),
	}
}

// flattenRoleRef produces the role_ref block from a Kubernetes RoleRef.
func flattenRoleRef(in api.RoleRef) []RoleRefModel {
	return []RoleRefModel{{
		APIGroup: types.StringValue(in.APIGroup),
		Kind:     types.StringValue(in.Kind),
		Name:     types.StringValue(in.Name),
	}}
}

// expandSubjects builds the list of Kubernetes Subjects from the subject blocks.
func expandSubjects(in []SubjectModel) []api.Subject {
	if len(in) == 0 {
		return []api.Subject{}
	}
	subjects := make([]api.Subject, 0, len(in))
	for _, s := range in {
		subject := api.Subject{
			Kind: s.Kind.ValueString(),
			Name: s.Name.ValueString(),
		}
		if !s.APIGroup.IsNull() && !s.APIGroup.IsUnknown() {
			subject.APIGroup = s.APIGroup.ValueString()
		}
		if !s.Namespace.IsNull() && !s.Namespace.IsUnknown() {
			subject.Namespace = s.Namespace.ValueString()
		}
		subjects = append(subjects, subject)
	}
	return subjects
}

// flattenSubjects produces the subject blocks from the list of Kubernetes Subjects.
func flattenSubjects(in []api.Subject) []SubjectModel {
	subjects := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		model := SubjectModel{
			// api_group is Optional+Computed; store the concrete value
			// (empty for ServiceAccount subjects) to mirror the SDKv2 state.
			APIGroup: types.StringValue(s.APIGroup),
			Kind:     types.StringValue(s.Kind),
			Name:     types.StringValue(s.Name),
		}
		if s.Namespace != "" {
			model.Namespace = types.StringValue(s.Namespace)
		} else {
			// Preserve the SDKv2 "default" default so cluster-scoped
			// (User/Group) subjects that omit a namespace do not drift.
			model.Namespace = types.StringValue("default")
		}
		subjects = append(subjects, model)
	}
	return subjects
}
