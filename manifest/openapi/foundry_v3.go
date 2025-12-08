// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
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
	f := &foapiv3{doc: oapi3}

	err = f.buildGvkIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to build GVK index when creating new foundry: %w", err)
	}

	return f, nil
}

func CRDSchemaToSpec(gvk schema.GroupVersionKind, crschema map[string]any) map[string]any {

	schema := make(map[string]any)
	for k, v := range crschema {
		schema[k] = v
	}
	schema["x-kubernetes-group-version-kind"] = []map[string]any{
		{
			"group":   gvk.Group,
			"version": gvk.Version,
			"kind":    gvk.Kind,
		},
	}
	return map[string]any{
		"openapi": "3.0",
		"info": map[string]any{
			"title":   "CRD schema wrapper",
			"version": "1.0.0",
		},
		"paths": map[string]any{},
		"components": map[string]any{
			"schemas": map[string]any{
				"crd-schema": schema,
			},
		},
	}
}

type foapiv3 struct {
	doc       *openapi3.T
	gkvIndex  sync.Map
}

func (f *foapiv3) GetTypeByGVK(gvk schema.GroupVersionKind) (tftypes.Type, map[string]string, error) {
	var hints map[string]string = make(map[string]string)

	// the ID string that OpenAPI uses to identify the resource
	// e.g. "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
	id, ok := f.gkvIndex.Load(gvk)
	if !ok {
		return nil, nil, fmt.Errorf("resource not found in OpenAPI index")
	}

	sref := f.doc.Components.Schemas[id.(string)]
	sch, err := resolveSchemaRef(sref, f.doc.Components.Schemas)
	if err != nil {
		return nil, hints, fmt.Errorf("failed to resolve schema: %w", err)
	}

	tftype, err := getTypeFromSchema(sch, 50, f.doc.Components.Schemas, tftypes.AttributePath{}, hints)
	return tftype, hints, err
}

// buildGvkIndex builds the reverse lookup index that associates each GVK
// to its corresponding string key in the swagger.Definitions map
func (f *foapiv3) buildGvkIndex() error {
	for did, dRef := range f.doc.Components.Schemas {
		def, err := resolveSchemaRef(dRef, f.doc.Components.Schemas)
		if err != nil {
			return err
		}
		ex, ok := def.Extensions["x-kubernetes-group-version-kind"]
		if !ok {
			continue
		}
		gvk := []schema.GroupVersionKind{}
		err = json.Unmarshal(([]byte)(ex.(json.RawMessage)), &gvk)
		if err != nil {
			return fmt.Errorf("failed to unmarshall GVK from OpenAPI schema extention: %w", err)
		}
		for i := range gvk {
			f.gkvIndex.Store(gvk[i], did)
		}
	}
	return nil
}
