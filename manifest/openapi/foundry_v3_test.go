// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
	"testing"
)

func TestNewFoundryFromSpecV3(t *testing.T) {
	sampleSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type": "string",
			},
			"bar": map[string]interface{}{
				"type": "number",
			},
		},
		"type": "object",
	}

	spec := SchemaToSpec("com.hashicorp.v1.TestCrd", sampleSchema)
	j, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Error: %+v", err)
	}

	f, err := NewFoundryFromSpecV3(j)
	if err != nil {
		t.Fatalf("Error: %+v", err)
	}

	if f.(*foapiv3).doc == nil {
		t.Fail()
	}
	if f.(*foapiv3).doc.Components.Schemas == nil {
		t.Fail()
	}
	crd, ok := f.(*foapiv3).doc.Components.Schemas["com.hashicorp.v1.TestCrd"]
	if !ok {
		t.Fail()
	}
	if crd == nil || crd.Value == nil {
		t.Fail()
	}
	if crd.Value.Type != "object" {
		t.Fail()
	}
	if crd.Value.Properties == nil {
		t.Fail()
	}
	foo, ok := crd.Value.Properties["foo"]
	if !ok {
		t.Fail()
	}
	if foo.Value.Type != "string" {
		t.Fail()
	}
	bar, ok := crd.Value.Properties["bar"]
	if !ok {
		t.Fail()
	}
	if bar.Value.Type != "number" {
		t.Fail()
	}
}
