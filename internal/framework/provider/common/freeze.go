// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FreezeSchema returns a copy of s in which every nested block carries its
// derived type in CustomType. Otherwise the framework rebuilds block types for
// every value path it visits, which is quadratic in the size of the pod spec.
// Maps are copied rather than mutated because schema constructors share them.
func FreezeSchema(s schema.Schema) schema.Schema {
	s.Blocks = freezeBlocks(s.Blocks)
	return s
}

func freezeBlocks(blocks map[string]schema.Block) map[string]schema.Block {
	if blocks == nil {
		return nil
	}
	out := make(map[string]schema.Block, len(blocks))
	for name, block := range blocks {
		out[name] = freezeBlock(block)
	}
	return out
}

func freezeBlock(block schema.Block) schema.Block {
	switch b := block.(type) {
	case schema.ListNestedBlock:
		return FreezeListNestedBlock(b)
	case schema.SetNestedBlock:
		b.NestedObject = FreezeNestedBlockObject(b.NestedObject)
		if b.CustomType == nil {
			b.CustomType = types.SetType{ElemType: b.NestedObject.Type()}
		}
		return b
	case schema.SingleNestedBlock:
		b.Blocks = freezeBlocks(b.Blocks)
		if b.CustomType == nil {
			b.CustomType = b.Type().(types.ObjectType)
		}
		return b
	default:
		return block
	}
}

// FreezeListNestedBlock freezes a single list block; see FreezeSchema.
func FreezeListNestedBlock(b schema.ListNestedBlock) schema.ListNestedBlock {
	b.NestedObject = FreezeNestedBlockObject(b.NestedObject)
	if b.CustomType == nil {
		b.CustomType = types.ListType{ElemType: b.NestedObject.Type()}
	}
	return b
}

// FreezeNestedBlockObject freezes the object of a list or set block; see FreezeSchema.
func FreezeNestedBlockObject(o schema.NestedBlockObject) schema.NestedBlockObject {
	o.Blocks = freezeBlocks(o.Blocks)
	if o.CustomType == nil {
		o.CustomType = o.Type()
	}
	return o
}

// SchemaFunc is the signature of resource.Resource.Schema.
type SchemaFunc func(context.Context, resource.SchemaRequest, *resource.SchemaResponse)

// FrozenSchema runs a request-independent schema builder once per process and
// shares the frozen result read-only.
func FrozenSchema(build SchemaFunc) SchemaFunc {
	once := sync.OnceValue(func() resource.SchemaResponse {
		var resp resource.SchemaResponse
		build(context.Background(), resource.SchemaRequest{}, &resp)
		resp.Schema = FreezeSchema(resp.Schema)
		return resp
	})
	return func(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
		frozen := once()
		resp.Schema = frozen.Schema
		resp.Diagnostics.Append(frozen.Diagnostics...)
	}
}
