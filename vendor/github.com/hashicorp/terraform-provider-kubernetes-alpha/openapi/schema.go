package openapi

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/mitchellh/hashstructure"
)

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
	case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
		t := openapi3.Schema{
			Type: "",
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps":
		t := openapi3.Schema{
			Type: "",
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.JSONSchemaProps":
		t := openapi3.Schema{
			Type: "",
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.CustomResourceSubresourceStatus":
		t := openapi3.Schema{
			Type: "object",
			AdditionalProperties: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus":
		t := openapi3.Schema{
			Type: "object",
			AdditionalProperties: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceDefinitionSpec":
		t, err := resolveSchemaRef(nref, defs)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve schema: %s", err)
		}
		vs := t.Properties["versions"]
		vs.Value.AdditionalProperties = vs.Value.Items
		vs.Value.Items = nil
		return t, nil
	}

	return resolveSchemaRef(nref, defs)
}

func getTypeFromSchema(elem *openapi3.Schema, stackdepth uint64, typeCache *sync.Map, defs map[string]*openapi3.SchemaRef) (tftypes.Type, error) {
	if stackdepth == 0 {
		// this is a hack to overcome the inability to express recursion in tftypes
		return nil, errors.New("recursion runaway while generating type from OpenAPI spec")
	}

	if elem == nil {
		return nil, errors.New("cannot convert OpenAPI type (nil)")
	}

	h, herr := hashstructure.Hash(elem, nil)

	var t tftypes.Type

	// check if type is in cache
	if herr == nil {
		if t, ok := typeCache.Load(h); ok {
			return t.(tftypes.Type), nil
		}
	}
	switch elem.Type {
	case "string":
		return tftypes.String, nil

	case "boolean":
		return tftypes.Bool, nil

	case "number":
		return tftypes.Number, nil

	case "integer":
		return tftypes.Number, nil

	case "":
		return tftypes.DynamicPseudoType, nil

	case "array":
		switch {
		case elem.Items != nil && elem.AdditionalProperties == nil: // normal array - translates to a tftypes.List
			it, err := resolveSchemaRef(elem.Items, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			et, err := getTypeFromSchema(it, stackdepth-1, typeCache, defs)
			if err != nil {
				return nil, err
			}
			t = tftypes.List{ElementType: et}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil
		case elem.AdditionalProperties != nil && elem.Items == nil: // "overriden" array - translates to a tftypes.List
			it, err := resolveSchemaRef(elem.AdditionalProperties, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			et, err := getTypeFromSchema(it, stackdepth-1, typeCache, defs)
			if err != nil {
				return nil, err
			}
			t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			if herr == nil {
				typeCache.Store(h, t)
			}
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
					return nil, fmt.Errorf("failed to resolve schema: %s", err)
				}
				pType, err := getTypeFromSchema(schema, stackdepth-1, typeCache, defs)
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

		case elem.Properties == nil && elem.AdditionalProperties != nil:
			// this is how OpenAPI defines associative arrays
			s, err := resolveSchemaRef(elem.AdditionalProperties, defs)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema: %s", err)
			}
			pt, err := getTypeFromSchema(s, stackdepth-1, typeCache, defs)
			if err != nil {
				return nil, err
			}
			t = tftypes.Map{AttributeType: pt}
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties == nil:
			// this is a strange case, encountered with io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1 and also io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus
			t = tftypes.DynamicPseudoType
			if herr == nil {
				typeCache.Store(h, t)
			}
			return t, nil

		}
	}

	return nil, fmt.Errorf("unknown type: %s", elem.Type)
}
