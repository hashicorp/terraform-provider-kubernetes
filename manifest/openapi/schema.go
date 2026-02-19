// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest"
	// "github.com/mitchellh/hashstructure"
)

func resolveSchemaRef(ref *openapi3.SchemaRef, defs map[string]*openapi3.SchemaRef) (*openapi3.Schema, error) {

	flattenedRef := ref
	if ref.Value != nil {
		if len(ref.Value.AllOf) == 1 &&
			combinationSchemaCount(ref.Value) == 1 &&
			len(ref.Value.Properties) == 0 &&
			ref.Value.AdditionalProperties == nil {

			flattenedRef = ref.Value.AllOf[0]
		}
	}

	sid := flattenedRef.Ref[strings.LastIndex(flattenedRef.Ref, "/")+1:]

	// These are exceptional situations that require non-standard types and that must be
	// handled first to not cause runaway recursion.
	switch sid {
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps":
		fallthrough
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.JSONSchemaProps":
		return &openapi3.Schema{Type: ""}, nil
	}

	if flattenedRef.Value != nil {
		return flattenedRef.Value, nil
	}

	nref, ok := defs[sid]
	if !ok {
		return nil, errors.New("schema not found")
	}
	if nref == nil {
		return nil, errors.New("nil schema reference")
	}

	return resolveSchemaRef(nref, defs)
}

func getTypeFromSchema(elem *openapi3.Schema, stackdepth uint64, defs map[string]*openapi3.SchemaRef, ap tftypes.AttributePath, th map[string]string) (tftypes.Type, error) {
	if stackdepth == 0 {
		// this is a hack to overcome the inability to express recursion in tftypes
		return nil, errors.New("recursion runaway while generating type from OpenAPI spec")
	}

	if elem == nil {
		return nil, errors.New("cannot convert OpenAPI type (nil)")
	}

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

	switch elem.Type {
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
		if elem.Format == "int-or-string" {
			th[ap.String()] = "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
			return tftypes.String, nil
		}
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
		// Check if it is just a union of primitives, and if this is the case try to translate it to a suitable tftypes primitive.
		if len(elem.OneOf) > 0 &&
			len(elem.OneOf) == combinationSchemaCount(elem) &&
			len(elem.Properties) == 0 &&
			elem.AdditionalProperties == nil {

			var stringUnion, intUnion, numberUnion, boolUnion, otherUnion bool

			for _, oneOfRef := range elem.OneOf {
				oneOfSchema, err := resolveSchemaRef(oneOfRef, defs)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve schema for OenOf items: %w", err)
				}
				oneOfTftype, err := getTypeFromSchema(oneOfSchema, stackdepth-1, defs, ap, th)
				if err != nil {
					return nil, err
				}

				switch {
				case oneOfTftype.Is(tftypes.String):
					stringUnion = true
				case oneOfTftype.Is(tftypes.Number):
					if oneOfSchema.Type == "integer" {
						intUnion = true
					} else {
						numberUnion = true
					}
				case oneOfTftype.Is(tftypes.Bool):
					boolUnion = true
				default:
					otherUnion = true
				}
			}

			switch {
			case otherUnion: // OneOf contained something that couldn't be translated to a fully knowns primitive
				break
			case stringUnion: // A union of string and any other primitives can always be mapped to string
				if intUnion && !numberUnion && !boolUnion {
					th[ap.String()] = "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
				}
				return tftypes.String, nil
			case intUnion || numberUnion: // oapi number and integer are both mapped to number
				if !boolUnion {
					return tftypes.Number, nil
				}
			case boolUnion: // Only bool
				return tftypes.Bool, nil
			}
		}

		return tftypes.DynamicPseudoType, nil // this is where DynamicType is set for when an attribute is tagged as 'x-kubernetes-preserve-unknown-fields'

	case "array":
		switch {
		case elem.Items != nil && elem.AdditionalProperties == nil: // normal array - translates to a tftypes.List
			it, err := resolveSchemaRef(elem.Items, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %w", err)
			}
			aap := ap.WithElementKeyInt(-1)
			et, err := getTypeFromSchema(it, stackdepth-1, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			if !isTypeFullyKnown(et) {
				t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			} else {
				t = tftypes.List{ElementType: et}
			}
			return t, nil
		case elem.AdditionalProperties != nil && elem.Items == nil: // "overriden" array - translates to a tftypes.Tuple
			it, err := resolveSchemaRef(elem.AdditionalProperties, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %w", err)
			}
			aap := ap.WithElementKeyInt(-1)
			et, err := getTypeFromSchema(it, stackdepth-1, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			return t, nil
		}

	case "object":

		switch {
		case elem.Properties != nil && elem.AdditionalProperties == nil:
			// this is a standard OpenAPI object
			atts := make(map[string]tftypes.Type, len(elem.Properties))
			for p, v := range elem.Properties {
				schema, err := resolveSchemaRef(v, defs)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve schema: %w", err)
				}
				aap := ap.WithAttributeName(p)
				pType, err := getTypeFromSchema(schema, stackdepth-1, defs, *aap, th)
				if err != nil {
					return nil, err
				}
				atts[p] = pType
			}
			t = tftypes.Object{AttributeTypes: atts}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties != nil:
			// this is how OpenAPI defines associative arrays
			s, err := resolveSchemaRef(elem.AdditionalProperties, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema: %w", err)
			}
			aap := ap.WithElementKeyString("#")
			pt, err := getTypeFromSchema(s, stackdepth-1, defs, *aap, th)
			if err != nil {
				return nil, err
			}
			t = tftypes.Map{ElementType: pt}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties == nil:
			// this is a strange case, encountered with io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1 and also io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus
			t = tftypes.DynamicPseudoType
			return t, nil

		}
	}

	return nil, fmt.Errorf("unknown type: %w", elem.Type)
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

func combinationSchemaCount(schema *openapi3.Schema) int {
	notCount := 0
	if schema.Not != nil {
		notCount = 1
	}
	return notCount + len(schema.AllOf) + len(schema.AnyOf) + len(schema.OneOf)
}
