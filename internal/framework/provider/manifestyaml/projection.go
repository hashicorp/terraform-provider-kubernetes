// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

// projectOwned reduces a live/dry-run object to only the fields owned by our
// SSA field manager, minus ignore_fields, and returns canonical JSON. This is
// the drift anchor stored in `live_manifest` (RFC-011 §6.3): plans compare only
// owned fields, so server defaults / other managers' fields never cause drift.
func projectOwned(obj *unstructured.Unstructured, fieldManager string, ignore []string) (string, error) {
	owned := &fieldpath.Set{}
	found := false
	for _, e := range obj.GetManagedFields() {
		// Only our SSA "Apply" entries; ignore "Update" (non-SSA) and the status subresource.
		if e.Manager != fieldManager || e.Operation != metav1.ManagedFieldsOperationApply || e.Subresource != "" {
			continue
		}
		if e.FieldsV1 == nil || len(e.FieldsV1.Raw) == 0 {
			continue
		}
		s := &fieldpath.Set{}
		if err := s.FromJSON(bytes.NewReader(e.FieldsV1.Raw)); err != nil {
			return "", err
		}
		owned = owned.Union(s)
		found = true
	}

	projected := map[string]interface{}{}
	if found {
		tv, err := typed.DeducedParseableType.FromUnstructured(obj.Object)
		if err != nil {
			return "", err
		}
		if m, ok := tv.ExtractItems(owned).AsValue().Unstructured().(map[string]interface{}); ok {
			projected = m
		}
	}

	for _, p := range ignore {
		removeDotPath(projected, p)
	}

	// encoding/json sorts map keys → canonical, stable for comparison.
	b, err := json.Marshal(projected)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// removeDotPath deletes a dotted path (e.g. "spec.replicas") from a nested map.
func removeDotPath(m map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			delete(cur, p)
			return
		}
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			return
		}
		cur = next
	}
}

// ignoreList extracts ignore_fields from the model as []string.
func ignoreList(ctx context.Context, m *manifestYAMLModel) []string {
	if m.IgnoreFields.IsNull() || m.IgnoreFields.IsUnknown() {
		return nil
	}
	var out []string
	m.IgnoreFields.ElementsAs(ctx, &out, false)
	return out
}

// fieldManagerOf returns the effective field manager (default "terraform").
func fieldManagerOf(m *manifestYAMLModel) string {
	if m.FieldManager.IsNull() || m.FieldManager.IsUnknown() || m.FieldManager.ValueString() == "" {
		return "terraform"
	}
	return m.FieldManager.ValueString()
}

// setLiveManifest projects owned fields from obj and writes live_manifest on the model.
func setLiveManifest(ctx context.Context, m *manifestYAMLModel, obj *unstructured.Unstructured) error {
	proj, err := projectOwned(obj, fieldManagerOf(m), ignoreList(ctx, m))
	if err != nil {
		return err
	}
	m.LiveManifest = types.StringValue(proj)
	return nil
}
