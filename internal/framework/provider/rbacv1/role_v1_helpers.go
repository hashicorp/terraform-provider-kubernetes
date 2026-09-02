// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ID helpers, matching the namespace/name format used by the SDKv2 resource.

func buildID(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func parseID(id string) (namespace, name string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
	}
	return parts[0], parts[1], nil
}

// String map helpers (labels, annotations).

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

// String set helpers (api_groups, resources, resource_names, verbs).

func expandStringSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var values []types.String
	diags := set.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, diags
	}

	result := make([]string, 0, len(values))
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result, nil
}

func flattenStringSet(slice []string) (types.Set, diag.Diagnostics) {
	if len(slice) == 0 {
		return types.SetNull(types.StringType), nil
	}

	elements := make([]attr.Value, len(slice))
	for i, v := range slice {
		elements[i] = types.StringValue(v)
	}

	return types.SetValue(types.StringType, elements)
}

// Metadata helpers.

func expandMetadata(m MetadataModel) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Annotations:  expandStringMap(m.Annotations),
		Labels:       expandStringMap(m.Labels),
		GenerateName: m.GenerateName.ValueString(),
		Name:         m.Name.ValueString(),
		Namespace:    m.Namespace.ValueString(),
	}
}

// flattenMetadata converts a Kubernetes ObjectMeta into MetadataModel,
// filtering internal Kubernetes-managed keys and user-configured
// ignore_annotations/ignore_labels patterns out of annotations/labels —
// unless the key is already present in `current` (i.e. explicitly managed
// by this Terraform config), matching SDKv2's flattenMetadata
// (kubernetes/structures.go: removeInternalKeys/removeKeys). `current` is
// the metadata already known to Terraform before this read (the prior
// state in Read, the plan in Create/Update); pass a zero MetadataModel
// (nothing yet managed) when there is none, such as during ImportState.
func flattenMetadata(meta metav1.ObjectMeta, current MetadataModel, ignoreAnnotations, ignoreLabels []string) *MetadataModel {
	m := &MetadataModel{
		Generation:      types.Int64Value(meta.Generation),
		Name:            types.StringValue(meta.Name),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
		Annotations:     flattenStringMap(filterManagedMetadataKeys(meta.Annotations, current.Annotations, ignoreAnnotations)),
		Labels:          flattenStringMap(filterManagedMetadataKeys(meta.Labels, current.Labels, ignoreLabels)),
	}
	if meta.GenerateName != "" {
		m.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		m.GenerateName = types.StringNull()
	}
	if meta.Namespace != "" {
		m.Namespace = types.StringValue(meta.Namespace)
	}
	return m
}

// filterManagedMetadataKeys drops keys from `m` that look like internal
// Kubernetes keys or match an ignore_annotations/ignore_labels pattern,
// unless that key is already present in `current` (already managed by this
// Terraform config, so keep tracking it regardless of the patterns above).
func filterManagedMetadataKeys(m map[string]string, current map[string]types.String, ignorePatterns []string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if _, managed := current[k]; !managed && (isInternalMetadataKey(k) || matchesIgnorePattern(k, ignorePatterns)) {
			continue
		}
		result[k] = v
	}
	return result
}

// isInternalMetadataKey mirrors kubernetes.isInternalKey (kubernetes/structures.go).
func isInternalMetadataKey(key string) bool {
	u, err := url.Parse("//" + key)
	if err != nil {
		return false
	}

	// allow user specified application specific keys
	if u.Hostname() == "app.kubernetes.io" {
		return false
	}

	// allow AWS load balancer configuration annotations
	if u.Hostname() == "service.beta.kubernetes.io" {
		return false
	}

	// internal *.kubernetes.io keys
	if strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}

	// Specific to DaemonSet annotations, generated & controlled by the server.
	if strings.Contains(key, "deprecated.daemonset.template.generation") {
		return true
	}
	return false
}

// matchesIgnorePattern mirrors kubernetes.ignoreKey (kubernetes/structures.go).
func matchesIgnorePattern(key string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := regexp.MatchString(p, key); ok {
			return true
		}
	}
	return false
}

// PolicyRule helpers.

func expandPolicyRules(ctx context.Context, rules []RuleModel) ([]rbacv1api.PolicyRule, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	result := make([]rbacv1api.PolicyRule, len(rules))

	for i, rule := range rules {
		apiGroups, diags := expandStringSet(ctx, rule.APIGroups)
		allDiags.Append(diags...)

		resources, diags := expandStringSet(ctx, rule.Resources)
		allDiags.Append(diags...)

		resourceNames, diags := expandStringSet(ctx, rule.ResourceNames)
		allDiags.Append(diags...)

		verbs, diags := expandStringSet(ctx, rule.Verbs)
		allDiags.Append(diags...)

		result[i] = rbacv1api.PolicyRule{
			APIGroups:     apiGroups,
			Resources:     resources,
			ResourceNames: resourceNames,
			Verbs:         verbs,
		}
	}

	return result, allDiags
}

func flattenPolicyRules(rules []rbacv1api.PolicyRule) ([]RuleModel, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	result := make([]RuleModel, len(rules))

	for i, rule := range rules {
		apiGroups, diags := flattenStringSet(rule.APIGroups)
		allDiags.Append(diags...)

		resources, diags := flattenStringSet(rule.Resources)
		allDiags.Append(diags...)

		resourceNames, diags := flattenStringSet(rule.ResourceNames)
		allDiags.Append(diags...)

		verbs, diags := flattenStringSet(rule.Verbs)
		allDiags.Append(diags...)

		result[i] = RuleModel{
			APIGroups:     apiGroups,
			Resources:     resources,
			ResourceNames: resourceNames,
			Verbs:         verbs,
		}
	}

	return result, allDiags
}

// rulesEqual reports whether two rule lists are identical, treating each
// rule's set-typed fields with set semantics (order-independent).
func rulesEqual(a, b []RuleModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].APIGroups.Equal(b[i].APIGroups) ||
			!a[i].Resources.Equal(b[i].Resources) ||
			!a[i].ResourceNames.Equal(b[i].ResourceNames) ||
			!a[i].Verbs.Equal(b[i].Verbs) {
			return false
		}
	}
	return true
}

// JSON-patch helpers for Update, mirroring kubernetes.patchMetadata/diffStringMap
// so that the generated patches are identical to the SDKv2 implementation.

func buildMetadataPatch(plan, state MetadataModel) kubernetes.PatchOperations {
	var ops kubernetes.PatchOperations
	ops = append(ops, diffStringMap("/metadata/annotations", state.Annotations, plan.Annotations)...)
	ops = append(ops, diffStringMap("/metadata/labels", state.Labels, plan.Labels)...)
	return ops
}

func diffStringMap(pathPrefix string, oldV, newV map[string]types.String) kubernetes.PatchOperations {
	ops := make(kubernetes.PatchOperations, 0)
	pathPrefix = strings.TrimRight(pathPrefix, "/")

	if len(oldV) == 0 {
		if len(newV) == 0 {
			return ops
		}
		ops = append(ops, &kubernetes.AddOperation{
			Path:  pathPrefix,
			Value: expandStringMap(newV),
		})
		return ops
	}

	for k := range oldV {
		if _, ok := newV[k]; ok {
			continue
		}
		ops = append(ops, &kubernetes.RemoveOperation{
			Path: pathPrefix + "/" + escapeJSONPointer(k),
		})
	}

	for k, v := range newV {
		newValue := v.ValueString()

		if oldValue, ok := oldV[k]; ok {
			if oldValue.ValueString() == newValue {
				continue
			}
			ops = append(ops, &kubernetes.ReplaceOperation{
				Path:  pathPrefix + "/" + escapeJSONPointer(k),
				Value: newValue,
			})
			continue
		}

		ops = append(ops, &kubernetes.AddOperation{
			Path:  pathPrefix + "/" + escapeJSONPointer(k),
			Value: newValue,
		})
	}

	return ops
}

// escapeJSONPointer escapes a string per RFC 6901 so it can be used as a
// path segment in JSON patch operations.
func escapeJSONPointer(path string) string {
	path = strings.ReplaceAll(path, "~", "~0")
	path = strings.ReplaceAll(path, "/", "~1")
	return path
}
