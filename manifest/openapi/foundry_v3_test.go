// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
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
	gvk := schema.FromAPIVersionAndKind("hashicorp.com/v1", "TestCrd")
	spec := CRDSchemaToSpec(gvk, sampleSchema)
	j, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Error: %+v", err)
	}

	f, err := NewFoundryFromSpecV3(j)
	if err != nil {
		t.Fatalf("Error creating foundry: %v", err)
	}

	f3, ok := f.(*foapiv3)
	if !ok {
		t.Fatal("foundry not of expected type")
	}

	if f3.doc == nil {
		t.Fatal("no doc")
	}
	if f3.doc.Components.Schemas == nil {
		t.Fatal("no schemas")
	}
	id, ok := f3.gkvIndex.Load(gvk)
	if !ok {
		t.Fatal("could not lookup schema id")
	}
	crd, ok := f3.doc.Components.Schemas[id.(string)]
	if !ok {
		t.Fatal("CRD schema not found")
	}
	if crd == nil || crd.Value == nil {
		t.Fatal("CRD schema empty")
	}
	if crd.Value.Type != "object" {
		t.Fatal("CRD type not object")
	}
	if crd.Value.Properties == nil {
		t.Fatal("CRD missing properties")
	}
	foo, ok := crd.Value.Properties["foo"]
	if !ok {
		t.Fatal("CRD missing property foo")
	}
	if foo.Value.Type != "string" {
		t.Fatal("CRD property foo not a string")
	}
	bar, ok := crd.Value.Properties["bar"]
	if !ok {
		t.Fatal("CRD missing property bar")
	}
	if bar.Value.Type != "number" {
		t.Fatal("CRD property bar not a number")
	}
}
