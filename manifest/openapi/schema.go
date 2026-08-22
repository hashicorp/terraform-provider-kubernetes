// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest"
	"github.com/mitchellh/hashstructure"
)

// Helper functions to handle openapi2 vs openapi3 schema differences

// resolveSchemaRefV2 resolves openapi2 SchemaRef to Schema
func resolveSchemaRefV2(ref *openapi2.SchemaRef, defs map[string]*openapi2.SchemaRef) (*openapi2.Schema, error) {
	if ref.Value != nil {
		return ref.Value, nil
	}

	rp := strings.Split(ref.Ref, "/")
	sid := rp[len(rp)-1]

	nref, ok := defs[sid]

	if !ok {
		return nil, errors.New("schema not found")
	}
	if nref == nil {
		return nil, errors.New("nil schema reference")
	}

	// These are exceptional situations that require non-standard types.
	switch sid {
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps":
		t := openapi2.Schema{
			Type: &openapi3.Types{},
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.JSONSchemaProps":
		t := openapi2.Schema{
			Type: &openapi3.Types{},
		}
		return &t, nil
	}

	return resolveSchemaRefV2(nref, defs)
}

// getTypeFromSchemaV2 converts openapi2 Schema to tftypes
func getTypeFromSchemaV2(elem *openapi2.Schema, stackdepth uint64, typeCache *sync.Map, defs map[string]*openapi2.SchemaRef, ap tftypes.AttributePath, th map[string]string) (tftypes.Type, error) {
	if stackdepth == 0 {
		return nil, errors.New("recursion runaway while generating type from OpenAPI spec")
	}

	if elem == nil {
		return nil, errors.New("cannot convert OpenAPI type (nil)")
	}

	h, herr := hashstructure.Hash(elem, nil)

	var t tftypes.Type

	// Check if attribute type is tagged as 'x-kubernetes-preserve-unknown-fields' in OpenAPI.
	if xpufJSON, ok := elem.Extensions[manifest.PreserveUnknownFieldsLabel]; ok {
		var xpuf bool
		v, err := xpufJSON.(json.RawMessage).MarshalJSON()
		if err == nil {
			err = json.Unmarshal(v, &xpuf)
			if err == nil && xpuf {
				th[ap.String()] = manifest.PreserveUnknownFieldsLabel
			}
		}
	}

	// Get first type from the Types slice
	typeStrV2 := getFirstSchemaType(elem.Type)

	switch typeStrV2 {
	case "string":
		if elem.Format == "int-or-string" {
			th[ap.String()] = "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
		}
		return tftypes.String, nil

	case "boolean":
		return tftypes.Bool, nil

	case "number":
		return tftypes.Number, nil

	case "integer":
		return tftypes.Number, nil

	case "":
		if xv, ok := elem.Extensions["x-kubernetes-int-or-string"]; ok {
			xb, err := xv.(json.RawMessage).MarshalJSON()
			if err != nil {
				return tftypes.DynamicPseudoType, nil
			}
			var x bool
			err = json.Unmarshal(xb, &x)
			if err == nil && x {
				th[ap.String()] = "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
				return tftypes.String, nil
			}
		}
		return tftypes.DynamicPseudoType, nil

	case "array":
		hasAdditionalPropsV2 := elem.AdditionalProperties.Schema != nil
		switch {
		case elem.Items != nil && !hasAdditionalPropsV2:
			it, err := resolveSchemaRefV2(elem.Items, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			aap := ap.WithElementKeyInt(-1)
			et, err := getTypeFromSchemaV2(it, stackdepth-1, typeCache, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			if !isTypeFullyKnown(et) {
				t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			} else {
				t = tftypes.List{ElementType: et}
			}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil
		case hasAdditionalPropsV2 && elem.Items == nil:
			// AdditionalProperties.Schema is *openapi3.SchemaRef, but we're in V2 context
			// This case seems unlikely with K8s OpenAPI specs, but handle it anyway
			// For now, treat as DynamicPseudoType
			return tftypes.DynamicPseudoType, nil
		}

	case "object":
		hasAdditionalPropsSchemaV2 := elem.AdditionalProperties.Schema != nil
		switch {
		case elem.Properties != nil && !hasAdditionalPropsSchemaV2:
			atts := make(map[string]tftypes.Type, len(elem.Properties))
			for p, v := range elem.Properties {
				schema, err := resolveSchemaRefV2(v, defs)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve schema: %s", err)
				}
				aap := ap.WithAttributeName(p)
				pType, err := getTypeFromSchemaV2(schema, stackdepth-1, typeCache, defs, *aap, th)
				if err != nil {
					return nil, err
				}
				atts[p] = pType
			}
			t = tftypes.Object{AttributeTypes: atts}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil

		case elem.Properties == nil && hasAdditionalPropsSchemaV2:
			// AdditionalProperties.Schema is *openapi3.SchemaRef in V2 context
			// This represents a map type - treat as DynamicPseudoType for simplicity
			return tftypes.DynamicPseudoType, nil

		case elem.Properties == nil && !hasAdditionalPropsSchemaV2:
			t = tftypes.DynamicPseudoType
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil
		}
	}

	return nil, fmt.Errorf("unknown type: %v", elem.Type)
}

// schemaTypeContains checks if the schema type contains a specific type string
func schemaTypeContains(schemaType *openapi3.Types, typeStr string) bool {
	if schemaType == nil {
		return typeStr == ""
	}
	for _, t := range *schemaType {
		if t == typeStr {
			return true
		}
	}
	return false
}

// getFirstSchemaType gets the first type from a schema Type field, or empty string
func getFirstSchemaType(schemaType *openapi3.Types) string {
	if schemaType == nil || len(*schemaType) == 0 {
		return ""
	}
	return (*schemaType)[0]
}

func resolveSchemaRef(ref *openapi3.SchemaRef, defs map[string]*openapi3.SchemaRef) (*openapi3.Schema, error) {
	if ref.Value != nil {
		return ref.Value, nil
	}

	rp := strings.Split(ref.Ref, "/")
	sid := rp[len(rp)-1]

	nref, ok := defs[sid]

	if !ok {
		return nil, errors.New("schema not found")
	}
	if nref == nil {
		return nil, errors.New("nil schema reference")
	}

	// These are exceptional situations that require non-standard types.
	switch sid {
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps":
		t := openapi3.Schema{
			Type: &openapi3.Types{},
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.JSONSchemaProps":
		t := openapi3.Schema{
			Type: &openapi3.Types{},
		}
		return &t, nil
	}

	return resolveSchemaRef(nref, defs)
}

func getTypeFromSchema(elem *openapi3.Schema, stackdepth uint64, typeCache *sync.Map, defs map[string]*openapi3.SchemaRef, ap tftypes.AttributePath, th map[string]string) (tftypes.Type, error) {
	if stackdepth == 0 {
		// this is a hack to overcome the inability to express recursion in tftypes
		return nil, errors.New("recursion runaway while generating type from OpenAPI spec")
	}

	if elem == nil {
		return nil, errors.New("cannot convert OpenAPI type (nil)")
	}

	h, herr := hashstructure.Hash(elem, nil)

	var t tftypes.Type

	// Check if attribute type is tagged as 'x-kubernetes-preserve-unknown-fields' in OpenAPI.
	// If so, we add a type hint to indicate this and return DynamicPseudoType for this attribute,
	// since we have no further structural information about it.
	if xpufJSON, ok := elem.Extensions[manifest.PreserveUnknownFieldsLabel]; ok {
		var xpuf bool
		v, err := xpufJSON.(json.RawMessage).MarshalJSON()
		if err == nil {
			err = json.Unmarshal(v, &xpuf)
			if err == nil && xpuf {
				th[ap.String()] = manifest.PreserveUnknownFieldsLabel
			}
		}
	}

	// check if type is in cache
	// HACK: this is temporarily disabled to diagnose a cache corruption issue.
	// if herr == nil {
	// 	if t, ok := typeCache.Load(h); ok {
	// 		return t.(tftypes.Type), nil
	// 	}
	// }

	// Get first type from the Types slice
	typeStr := getFirstSchemaType(elem.Type)

	switch typeStr {
	case "string":
		if elem.Format == "int-or-string" {
			th[ap.String()] = "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
		}
		return tftypes.String, nil

	case "boolean":
		return tftypes.Bool, nil

	case "number":
		return tftypes.Number, nil

	case "integer":
		return tftypes.Number, nil

	case "":
		if xv, ok := elem.Extensions["x-kubernetes-int-or-string"]; ok {
			xb, err := xv.(json.RawMessage).MarshalJSON()
			if err != nil {
				return tftypes.DynamicPseudoType, nil
			}
			var x bool
			err = json.Unmarshal(xb, &x)
			if err == nil && x {
				th[ap.String()] = "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
				return tftypes.String, nil
			}
		}
		return tftypes.DynamicPseudoType, nil // this is where DynamicType is set for when an attribute is tagged as 'x-kubernetes-preserve-unknown-fields'

	case "array":
		hasAdditionalProps := elem.AdditionalProperties.Schema != nil
		switch {
		case elem.Items != nil && !hasAdditionalProps: // normal array - translates to a tftypes.List
			it, err := resolveSchemaRef(elem.Items, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			aap := ap.WithElementKeyInt(-1)
			et, err := getTypeFromSchema(it, stackdepth-1, typeCache, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			if !isTypeFullyKnown(et) {
				t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			} else {
				t = tftypes.List{ElementType: et}
			}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil
		case hasAdditionalProps && elem.Items == nil: // "overriden" array - translates to a tftypes.Tuple
			it, err := resolveSchemaRef(elem.AdditionalProperties.Schema, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			aap := ap.WithElementKeyInt(-1)
			et, err := getTypeFromSchema(it, stackdepth-1, typeCache, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			return t, nil
		}

	case "object":
		hasAdditionalPropsSchema := elem.AdditionalProperties.Schema != nil
		switch {
		case elem.Properties != nil && !hasAdditionalPropsSchema:
			// this is a standard OpenAPI object
			atts := make(map[string]tftypes.Type, len(elem.Properties))
			for p, v := range elem.Properties {
				schema, err := resolveSchemaRef(v, defs)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve schema: %s", err)
				}
				aap := ap.WithAttributeName(p)
				pType, err := getTypeFromSchema(schema, stackdepth-1, typeCache, defs, *aap, th)
				if err != nil {
					return nil, err
				}
				atts[p] = pType
			}
			t = tftypes.Object{AttributeTypes: atts}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil

		case elem.Properties == nil && hasAdditionalPropsSchema:
			// this is how OpenAPI defines associative arrays
			s, err := resolveSchemaRef(elem.AdditionalProperties.Schema, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema: %s", err)
			}
			aap := ap.WithElementKeyString("#")
			pt, err := getTypeFromSchema(s, stackdepth-1, typeCache, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			t = tftypes.Map{ElementType: pt}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil

		case elem.Properties == nil && !hasAdditionalPropsSchema:
			// this is a strange case, encountered with io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1 and also io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus
			t = tftypes.DynamicPseudoType
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil

		}
	}

	return nil, fmt.Errorf("unknown type: %v", elem.Type)
}

func isTypeFullyKnown(t tftypes.Type) bool {
	if t.Is(tftypes.DynamicPseudoType) {
		return false
	}
	switch {
	case t.Is(tftypes.Object{}):
		for _, att := range t.(tftypes.Object).AttributeTypes {
			if !isTypeFullyKnown(att) {
				return false
			}
		}
	case t.Is(tftypes.Tuple{}):
		for _, ett := range t.(tftypes.Tuple).ElementTypes {
			if !isTypeFullyKnown(ett) {
				return false
			}
		}
	case t.Is(tftypes.List{}):
		return isTypeFullyKnown(t.(tftypes.List).ElementType)
	case t.Is(tftypes.Set{}):
		return isTypeFullyKnown(t.(tftypes.Set).ElementType)
	case t.Is(tftypes.Map{}):
		return isTypeFullyKnown(t.(tftypes.Map).ElementType)
	}
	return true
}
