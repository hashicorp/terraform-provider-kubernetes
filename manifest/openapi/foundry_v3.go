// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"fmt"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewFoundryFromSpecV3(spec []byte) (Foundry, error) {
	loader := openapi3.NewLoader()
	oapi3, err := loader.LoadFromData(spec)
	if err != nil {
		return nil, err
	}
	return &foapiv3{doc: oapi3}, nil
}

func SchemaToSpec(key string, crschema map[string]interface{}) map[string]interface{} {
	schema := make(map[string]interface{})
	for k, v := range crschema {
		schema[k] = v
	}
	return map[string]interface{}{
		"openapi": "3.0",
		"info": map[string]interface{}{
			"title":   "CRD schema wrapper",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				key: schema,
			},
		},
	}
}

type foapiv3 struct {
	doc       *openapi3.T
	gate      sync.Mutex
	typeCache sync.Map
}

func (f *foapiv3) GetTypeByGVK(_ schema.GroupVersionKind) (tftypes.Type, map[string]string, error) {
	f.gate.Lock()
	defer f.gate.Unlock()

	var hints map[string]string = make(map[string]string)
	ap := tftypes.AttributePath{}

	sref := f.doc.Components.Schemas[""]

	sch, err := resolveSchemaRef(sref, f.doc.Components.Schemas)
	if err != nil {
		return nil, hints, fmt.Errorf("failed to resolve schema: %s", err)
	}

	tftype, err := getTypeFromSchema(sch, 50, &(f.typeCache), f.doc.Components.Schemas, ap, hints)
	return tftype, hints, err
}
