// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var dnsLabelRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// expandStringMap converts the Framework typed string map to the plain map[string]string
// expected by Kubernetes ObjectMeta.
func expandStringMap(m map[string]types.String) map[string]string {
	if m == nil {
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

// buildRuntimeClassObject constructs the Kubernetes API struct from the plan model.
// Metadata is a slice of one (ListNestedBlock) — index [0] is always the element.
func buildRuntimeClassObject(plan RuntimeClassV1Model) *nodev1.RuntimeClass {
	return &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:            plan.Metadata[0].Name.ValueString(),
			GenerateName:    plan.Metadata[0].GenerateName.ValueString(),
			ResourceVersion: plan.Metadata[0].ResourceVersion.ValueString(),
			Labels:          expandStringMap(plan.Metadata[0].Labels),
			Annotations:     expandStringMap(plan.Metadata[0].Annotations),
		},
		Handler: plan.Handler.ValueString(),
	}
}

// flattenStringMap converts map[string]string (Kubernetes) to the typed map
// used in MetadataModel. Returns nil for empty maps.
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

// flattenMetadata reads back the server's ObjectMeta into the model.
// It strips cluster-managed annotation/label keys before writing to state
// to prevent permanent diffs from cluster-injected keys like
// kubectl.kubernetes.io/last-applied-configuration.
func flattenMetadata(
	meta metav1.ObjectMeta,
	configuredAnnotations map[string]types.String,
	configuredLabels map[string]types.String,
	ignoreAnnotations []string,
	ignoreLabels []string,
) MetadataModel {
	annotations := make(map[string]string)
	for k, v := range meta.Annotations {
		annotations[k] = v
	}
	removeInternalKeys(annotations, configuredAnnotations)
	removeKeys(annotations, configuredAnnotations, ignoreAnnotations)

	labels := make(map[string]string)
	for k, v := range meta.Labels {
		labels[k] = v
	}
	removeInternalKeys(labels, configuredLabels)
	removeKeys(labels, configuredLabels, ignoreLabels)

	m := MetadataModel{
		Name:            types.StringValue(meta.Name),
		UID:             types.StringValue(string(meta.UID)),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		Generation:      types.Int64Value(meta.Generation),
	}

	if meta.GenerateName != "" {
		m.GenerateName = types.StringValue(meta.GenerateName)
	}
	if len(annotations) > 0 {
		m.Annotations = flattenStringMap(annotations)
	}
	if len(labels) > 0 {
		m.Labels = flattenStringMap(labels)
	}

	return m
}

// removeKeys removes metadata keys that match any of the user-configured ignore
// patterns (regexes), unless the user has explicitly set that key in their config.
// Mirrors kubernetes/structures.go:removeKeys.
func removeKeys(m map[string]string, configuredKeys map[string]types.String, ignorePatterns []string) {
	for k := range m {
		if _, userOwns := configuredKeys[k]; userOwns {
			continue
		}
		for _, pattern := range ignorePatterns {
			if matched, _ := regexp.MatchString(pattern, k); matched {
				delete(m, k)
				break
			}
		}
	}
}

// removeInternalKeys removes cluster-managed keys from m unless the user
// has explicitly configured that key.
func removeInternalKeys(m map[string]string, configuredKeys map[string]types.String) {
	for k := range m {
		if _, userOwns := configuredKeys[k]; userOwns {
			continue
		}
		if isInternalKey(k) {
			delete(m, k)
		}
	}
}

// isInternalKey returns true when a metadata key is managed by the Kubernetes
// cluster itself and should not be stored in Terraform state.
func isInternalKey(annotationKey string) bool {
	u, err := url.Parse("//" + annotationKey)
	if err != nil {
		return false
	}
	if u.Hostname() == "app.kubernetes.io" {
		return false
	}
	if u.Hostname() == "service.beta.kubernetes.io" {
		return false
	}
	if strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}
	if strings.Contains(annotationKey, "deprecated.daemonset.template.generation") {
		return true
	}
	return false
}
