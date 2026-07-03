// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package kubemanifest holds the model-agnostic engine shared by the raw-manifest
// Framework resources (kubernetes_manifest_yaml and, per RFC-012,
// kubernetes_manifest_patch): YAML decoding, GVR resolution, owned-field projection
// (Server-Side Apply managedFields), and apply diagnostics. It contains no Terraform
// model types so it can be reused by any resource.
package kubemanifest

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
	"sigs.k8s.io/yaml"
)

// ProjectOwned reduces a live/dry-run object to only the fields owned by the given
// SSA field manager, minus the ignore paths, and returns canonical JSON. This is the
// drift anchor (RFC-011 §6.3): plans compare only owned fields, so server defaults /
// other managers' fields never cause drift.
//
// LIMITATION (RFC-011 §6.3.1): projection uses schemaless (deduced) typing, so LISTS
// ARE ATOMIC — owning any element projects the whole list. Fine-grained per-element
// extraction of associative lists needs the real OpenAPI schema (that is the domain of
// the typed kubernetes_manifest resource). Maps and scalars are granular.
func ProjectOwned(obj *unstructured.Unstructured, fieldManager string, ignore []string) (string, error) {
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
		RemoveDotPath(projected, p)
	}

	// encoding/json sorts map keys → canonical, stable for comparison.
	b, err := json.Marshal(projected)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RemoveDotPath deletes a dotted path (e.g. "spec.replicas") from a nested map.
func RemoveDotPath(m map[string]interface{}, path string) {
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

// ValueAtDotPath returns the value at a dotted path (e.g. "spec.serviceName") in a
// nested map, and whether it was present. Traversal stops (not found) at any
// non-map segment.
func ValueAtDotPath(m map[string]interface{}, dotted string) (interface{}, bool) {
	var cur interface{} = m
	for _, p := range strings.Split(dotted, ".") {
		cm, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, ok := cm[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// PathChanged reports whether the value at a dotted path differs between two objects
// (presence difference counts as a change).
func PathChanged(a, b map[string]interface{}, dotted string) bool {
	va, oka := ValueAtDotPath(a, dotted)
	vb, okb := ValueAtDotPath(b, dotted)
	if oka != okb {
		return true
	}
	return !reflect.DeepEqual(va, vb)
}

// NormalizeYAML converts a YAML document to canonical JSON (sorted keys), so that
// whitespace / comments / key-order differences do not register as changes.
func NormalizeYAML(y string) (string, error) {
	j, err := yaml.YAMLToJSON([]byte(y))
	if err != nil {
		return "", err
	}
	var v interface{}
	if err := json.Unmarshal(j, &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
