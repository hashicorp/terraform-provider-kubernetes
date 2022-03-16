package openapi

import (
	"encoding/json"
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
			Type:        "string",
			Description: "io.k8s.apimachinery.pkg.util.intstr.IntOrString", // this value later carries over as the "type hint"
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
	if xpufJSON, ok := elem.Extensions["x-kubernetes-preserve-unknown-fields"]; ok {
		var xpuf bool
		v, err := xpufJSON.(json.RawMessage).MarshalJSON()
		if err == nil {
			err = json.Unmarshal(v, &xpuf)
			if err == nil && xpuf {
				th[ap.String()] = "x-kubernetes-preserve-unknown-fields"
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
	switch elem.Type {
	case "string":
		switch elem.Description {
		case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
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
		switch {
		case elem.Items != nil && elem.AdditionalProperties == nil: // normal array - translates to a tftypes.List
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
		case elem.AdditionalProperties != nil && elem.Items == nil: // "overriden" array - translates to a tftypes.Tuple
			it, err := resolveSchemaRef(elem.AdditionalProperties, defs)
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

		switch {
		case elem.Properties != nil && elem.AdditionalProperties == nil:
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

		case elem.Properties == nil && elem.AdditionalProperties != nil:
			// this is how OpenAPI defines associative arrays
			s, err := resolveSchemaRef(elem.AdditionalProperties, defs)
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
