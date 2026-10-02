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

// FreezeSchema returns a copy of s in which every nested block, at any depth,
// carries its own derived attr.Type in CustomType.
//
// Without a CustomType, ListNestedBlock.Type(), SetNestedBlock.Type(),
// SingleNestedBlock.Type() and NestedBlockObject.Type() rebuild the type of the
// whole subtree on every call. The framework calls them through
// TypeAtTerraformPath for every prefix of every value path it visits
// (fromtftypes.AttributePath during nullify/reify/default/SetAtPath
// transforms), which is quadratic for the ~200-block pod spec. Precomputing
// the identical type makes each call O(1).
//
// The stored types are exactly the ones the framework would derive (they are
// computed with the framework's own Type() methods, bottom-up), so the wire
// schema, the values produced and plan behaviour are unchanged. An existing
// CustomType is never replaced. Maps are copied, never mutated, so constructors
// that share NestedBlockObject maps between parents remain safe.
func FreezeSchema(s schema.Schema) schema.Schema {
	s.Blocks = freezeBlocks(s.Blocks)
	return s
}

// freezeBlocks returns a copy of blocks with every block frozen; see FreezeSchema.
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

// freezeBlock returns block with its type, and the types of all nested blocks,
// precomputed; see FreezeSchema. Unknown block implementations are returned as-is.
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

// FreezeListNestedBlock freezes a single ListNestedBlock; see FreezeSchema.
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

// FrozenSchema wraps a resource schema builder so that it runs once per process
// and its result is frozen with FreezeSchema. The returned function is safe for
// concurrent use and hands every caller the same (read-only) schema. Builders
// must not depend on the request or context; resource schemas never do.
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
