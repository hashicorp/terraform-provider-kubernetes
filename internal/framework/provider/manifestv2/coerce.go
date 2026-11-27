// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// openAPISchema is the part of an OpenAPI v3 schema that says where a string
// belongs.
type openAPISchema struct {
	Type                 string                    `json:"type"`
	Ref                  string                    `json:"$ref"`
	AllOf                []*openAPISchema          `json:"allOf"`
	Properties           map[string]*openAPISchema `json:"properties"`
	Items                *openAPISchema            `json:"items"`
	AdditionalProperties json.RawMessage           `json:"additionalProperties"`
	IntOrString          bool                      `json:"x-kubernetes-int-or-string"`
	GroupVersionKinds    []schema.GroupVersionKind `json:"x-kubernetes-group-version-kind"`
}

type openAPIDocument struct {
	Components struct {
		Schemas map[string]*openAPISchema `json:"schemas"`
	} `json:"components"`
}

func (d *openAPIDocument) root(gvk schema.GroupVersionKind) *openAPISchema {
	for _, candidate := range d.Components.Schemas {
		for _, named := range candidate.GroupVersionKinds {
			if named == gvk {
				return candidate
			}
		}
	}
	return nil
}

// documents caches the OpenAPI document of each API version.
var documents sync.Map

// coerced returns the object with numbers and bools converted to strings
// where the kind's OpenAPI schema has a string, such as data = { PORT = 8080 }
// for a ConfigMap: HCL cannot tell a number that is meant as a string.
// Quantities and int-or-string fields keep their numbers. ok is false when
// nothing changed or the schema is not available.
func (r *ManifestV2) coerced(obj *unstructured.Unstructured) (*unstructured.Unstructured, bool) {
	gvk := obj.GroupVersionKind()
	var root *openAPISchema
	var document *openAPIDocument
	// A cached document may predate a CustomResourceDefinition of its version.
	for _, fresh := range []bool{false, true} {
		var err error
		if document, err = r.openAPIDocument(gvk.GroupVersion(), fresh); err != nil {
			return nil, false
		}
		if root = document.root(gvk); root != nil {
			break
		}
	}
	if root == nil {
		return nil, false
	}
	converted := obj.DeepCopy()
	changed := false
	coerce(converted.Object, root, document.Components.Schemas, 0, &changed)
	return converted, changed
}

func (r *ManifestV2) openAPIDocument(gv schema.GroupVersion, fresh bool) (*openAPIDocument, error) {
	key := "apis/" + gv.String()
	if gv.Group == "" {
		key = "api/" + gv.Version
	}
	if cached, ok := documents.Load(key); ok && !fresh {
		return cached.(*openAPIDocument), nil
	}
	clients, _, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	if diags.HasError() {
		return nil, fmt.Errorf("%s", diags[0].Detail())
	}
	disco, err := clients.DiscoveryClient()
	if err != nil {
		return nil, err
	}
	paths, err := disco.OpenAPIV3().Paths()
	if err != nil {
		return nil, err
	}
	path, found := paths[key]
	if !found {
		return nil, fmt.Errorf("the cluster publishes no OpenAPI schema for %s", gv)
	}
	raw, err := path.Schema("application/json")
	if err != nil {
		return nil, err
	}
	document := &openAPIDocument{}
	if err := json.Unmarshal(raw, document); err != nil {
		return nil, err
	}
	documents.Store(key, document)
	return document, nil
}

// resolve follows references, and an allOf that only wraps one.
func resolve(node *openAPISchema, schemas map[string]*openAPISchema) *openAPISchema {
	for range 32 {
		switch {
		case node == nil:
			return nil
		case node.Ref != "":
			node = schemas[strings.TrimPrefix(node.Ref, "#/components/schemas/")]
		case len(node.AllOf) == 1 && node.Type == "" && node.Properties == nil:
			node = node.AllOf[0]
		default:
			return node
		}
	}
	return nil
}

func coerce(value any, node *openAPISchema, schemas map[string]*openAPISchema, depth int, changed *bool) any {
	if node = resolve(node, schemas); node == nil || depth > 100 {
		return value
	}
	switch value := value.(type) {
	case map[string]any:
		var additional *openAPISchema
		if len(node.AdditionalProperties) > 0 && node.AdditionalProperties[0] == '{' {
			_ = json.Unmarshal(node.AdditionalProperties, &additional)
		}
		for name, member := range value {
			field := node.Properties[name]
			if field == nil {
				field = additional
			}
			value[name] = coerce(member, field, schemas, depth+1, changed)
		}
	case []any:
		for i := range value {
			value[i] = coerce(value[i], node.Items, schemas, depth+1, changed)
		}
	case int64, float64, bool:
		if node.Type != "string" || node.IntOrString {
			return value
		}
		*changed = true
		switch value := value.(type) {
		case int64:
			return strconv.FormatInt(value, 10)
		case float64:
			return strconv.FormatFloat(value, 'f', -1, 64)
		default:
			return strconv.FormatBool(value.(bool))
		}
	}
	return value
}
