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
	result := make(map[string]types.String, len(m))
	for k, v := range m {
		result[k] = types.StringValue(v)
	}
	return result
}

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
	elements := make([]attr.Value, len(slice))
	for i, v := range slice {
		elements[i] = types.StringValue(v)
	}

	return types.SetValue(types.StringType, elements)
}

func expandMetadata(m MetadataModel) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Annotations:  expandStringMap(m.Annotations),
		Labels:       expandStringMap(m.Labels),
		GenerateName: m.GenerateName.ValueString(),
		Name:         m.Name.ValueString(),
		Namespace:    m.Namespace.ValueString(),
	}
}

func flattenMetadata(meta metav1.ObjectMeta, current MetadataModel, ignoreAnnotations, ignoreLabels []string) *MetadataModel {
	m := &MetadataModel{
		Generation:      types.Int64Value(meta.Generation),
		Name:            types.StringValue(meta.Name),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
		Annotations:     flattenMetadataMap(meta.Annotations, current.Annotations, ignoreAnnotations),
		Labels:          flattenMetadataMap(meta.Labels, current.Labels, ignoreLabels),
	}
	m.GenerateName = types.StringValue(meta.GenerateName)
	if meta.Namespace != "" {
		m.Namespace = types.StringValue(meta.Namespace)
	}
	return m
}

func flattenMetadataMap(m map[string]string, current map[string]types.String, ignorePatterns []string) map[string]types.String {
	result := flattenStringMap(filterManagedMetadataKeys(m, current, ignorePatterns))
	// Null entries are configuration-only markers, not Kubernetes metadata.
	// Keep actual remote values when present so refresh can detect drift.
	for key, value := range current {
		if _, exists := result[key]; !exists && value.IsNull() {
			result[key] = value
		}
	}
	return result
}

func filterManagedMetadataKeys(m map[string]string, current map[string]types.String, ignorePatterns []string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		value, managed := current[k]
		if (!managed || value.IsNull()) && (isInternalMetadataKey(k) || matchesIgnorePattern(k, ignorePatterns)) {
			continue
		}
		result[k] = v
	}
	return result
}

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

func matchesIgnorePattern(key string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := regexp.MatchString(p, key); ok {
			return true
		}
	}
	return false
}

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

func buildMetadataPatch(plan, state MetadataModel) kubernetes.PatchOperations {
	var ops kubernetes.PatchOperations
	ops = append(ops, diffStringMap("/metadata/annotations", state.Annotations, plan.Annotations)...)
	ops = append(ops, diffStringMap("/metadata/labels", state.Labels, plan.Labels)...)
	return ops
}

func diffStringMap(pathPrefix string, oldV, newV map[string]types.String) kubernetes.PatchOperations {
	ops := make(kubernetes.PatchOperations, 0)
	pathPrefix = strings.TrimRight(pathPrefix, "/")
	oldValues := expandStringMap(oldV)
	newValues := expandStringMap(newV)

	if len(oldValues) == 0 {
		if len(newValues) == 0 {
			return ops
		}
		ops = append(ops, &kubernetes.AddOperation{
			Path:  pathPrefix,
			Value: newValues,
		})
		return ops
	}

	for k := range oldValues {
		if _, ok := newValues[k]; ok {
			continue
		}
		ops = append(ops, &kubernetes.RemoveOperation{
			Path: pathPrefix + "/" + escapeJSONPointer(k),
		})
	}

	for k, newValue := range newValues {
		if oldValue, ok := oldValues[k]; ok {
			if oldValue == newValue {
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

func escapeJSONPointer(path string) string {
	path = strings.ReplaceAll(path, "~", "~0")
	path = strings.ReplaceAll(path, "/", "~1")
	return path
}
