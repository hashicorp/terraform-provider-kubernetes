// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// This file holds the model-coupled projection helpers. The schemaless owned-field
// projection itself lives in package kubemanifest (see helpers.go delegations).

// ignoreList extracts ignore_fields from the model as []string.
func ignoreList(ctx context.Context, m *manifestYAMLModel) []string {
	if m.IgnoreFields.IsNull() || m.IgnoreFields.IsUnknown() {
		return nil
	}
	var out []string
	m.IgnoreFields.ElementsAs(ctx, &out, false)
	return out
}

// replaceOnList extracts force_replace_on from the model as []string.
func replaceOnList(ctx context.Context, m *manifestYAMLModel) []string {
	if m.ForceReplaceOn.IsNull() || m.ForceReplaceOn.IsUnknown() {
		return nil
	}
	var out []string
	m.ForceReplaceOn.ElementsAs(ctx, &out, false)
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

// setStatus writes the object's live .status as JSON into the status attribute.
func setStatus(m *manifestYAMLModel, obj *unstructured.Unstructured) error {
	st, found, err := unstructured.NestedFieldCopy(obj.Object, "status")
	if err != nil || !found || st == nil {
		m.Status = types.StringValue("{}")
		return nil
	}
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	m.Status = types.StringValue(string(b))
	return nil
}
