// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ObjectMetaGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ObjectMeta"}

// NewFoundryFromSpecV2 creates a new tftypes.Type foundry from an OpenAPI v2 spec document
// * spec argument should be a valid OpenAPI v2 JSON document
func NewFoundryFromSpecV2(spec []byte) (Foundry, error) {
	if len(spec) < 6 { // unlikely to be valid json
		return nil, errors.New("empty spec")
	}

	var swg openapi2.T
	err := swg.UnmarshalJSON(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %s", err)
	}

	d := swg.Definitions
	if len(d) == 0 {
		return nil, errors.New("spec has no type information")
	}

	f := foapiv2{
		swagger:        &swg,
		typeCache:      sync.Map{},
		gkvIndex:       sync.Map{}, //reverse lookup index from GVK to OpenAPI definition IDs
		recursionDepth: 50,         // arbitrarily large number - a type this deep will likely kill Terraform anyway
		gate:           sync.Mutex{},
	}

	err = f.buildGvkIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to build GVK index when creating new foundry: %s", err)
	}

	return &f, nil
}

// Foundry is a mechanism to construct tftypes out of OpenAPI specifications
type Foundry interface {
	GetTypeByGVK(gvk schema.GroupVersionKind) (tftypes.Type, map[string]string, error)
}

type foapiv2 struct {
	swagger        *openapi2.T
	typeCache      sync.Map
	gkvIndex       sync.Map
	recursionDepth uint64 // a last resort circuit-breaker for run-away recursion - hitting this will make for a bad day
	gate           sync.Mutex
}

// GetTypeByGVK looks up a type by its GVK in the Definitions sections of
// the OpenAPI spec and returns its (nearest) tftypes.Type equivalent
func (f *foapiv2) GetTypeByGVK(gvk schema.GroupVersionKind) (tftypes.Type, map[string]string, error) {
	f.gate.Lock()
	defer f.gate.Unlock()

	var hints map[string]string = make(map[string]string)
	ap := tftypes.AttributePath{}

	// ObjectMeta isn't discoverable via the index because it's not tagged with "x-kubernetes-group-version-kind" in OpenAPI spec
	// as top-level resouces schemas are. But we need ObjectMeta as a separate type when backfilling into CRD schemas.
	if gvk == ObjectMetaGVK {
		t, err := f.getTypeByID("io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", hints, ap)
		return t, hints, err
	}

	// the ID string that Swagger / OpenAPI uses to identify the resource
	// e.g. "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
	id, ok := f.gkvIndex.Load(gvk)
	if !ok {
		return nil, nil, fmt.Errorf("%v resource not found in OpenAPI index", gvk)
	}
	t, err := f.getTypeByID(id.(string), hints, ap)
	return t, hints, err
}

func (f *foapiv2) getTypeByID(id string, h map[string]string, ap tftypes.AttributePath) (tftypes.Type, error) {
	swd, ok := f.swagger.Definitions[id]

	if !ok {
		return nil, errors.New("invalid type identifier")
	}

	if swd == nil {
		return nil, errors.New("invalid type reference (nil)")
	}

	sch, err := resolveSchemaRef(swd, f.swagger.Definitions)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schema: %s", err)
	}

	return getTypeFromSchema(sch, f.recursionDepth, &(f.typeCache), f.swagger.Definitions, ap, h)
}

// buildGvkIndex builds the reverse lookup index that associates each GVK
// to its corresponding string key in the swagger.Definitions map
func (f *foapiv2) buildGvkIndex() error {
	for did, dRef := range f.swagger.Definitions {
		def, err := resolveSchemaRef(dRef, f.swagger.Definitions)
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
			return fmt.Errorf("failed to unmarshall GVK from OpenAPI schema extention: %v", err)
		}
		for i := range gvk {
			f.gkvIndex.Store(gvk[i], did)
		}
	}
	return nil
}
