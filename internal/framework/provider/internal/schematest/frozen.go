// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schematest

import (
	"bytes"
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// UnfrozenBlockPaths lists, sorted, the nested blocks under blocks that have
// no precomputed type (see common.FreezeSchema).
func UnfrozenBlockPaths(blocks map[string]schema.Block) []string {
	var out []string
	var walk func(prefix string, blocks map[string]schema.Block)
	walk = func(prefix string, blocks map[string]schema.Block) {
		for name, block := range blocks {
			key := prefix + name
			switch b := block.(type) {
			case schema.ListNestedBlock:
				if b.CustomType == nil || b.NestedObject.CustomType == nil {
					out = append(out, key)
				}
				walk(key+".", b.NestedObject.Blocks)
			case schema.SetNestedBlock:
				if b.CustomType == nil || b.NestedObject.CustomType == nil {
					out = append(out, key)
				}
				walk(key+".", b.NestedObject.Blocks)
			case schema.SingleNestedBlock:
				if b.CustomType == nil {
					out = append(out, key)
				}
				walk(key+".", b.Blocks)
			}
		}
	}
	walk("", blocks)
	sort.Strings(out)
	return out
}

// AssertFrozenSchemaUnchanged checks that the schema r serves has every nested
// block frozen, and that freezing changed nothing Terraform can observe: the
// schema type and the GetProviderSchema response must equal those of the
// schema produced by build, the unfrozen builder behind r.Schema.
func AssertFrozenSchemaUnchanged(t testing.TB, r resource.Resource, build func(context.Context, resource.SchemaRequest, *resource.SchemaResponse)) {
	t.Helper()
	ctx := context.Background()
	var frozen, built resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &frozen)
	build(ctx, resource.SchemaRequest{}, &built)
	if paths := UnfrozenBlockPaths(frozen.Schema.Blocks); len(paths) != 0 {
		t.Fatalf("blocks without a precomputed type: %v", paths)
	}
	if !frozen.Schema.Type().Equal(built.Schema.Type()) {
		t.Fatal("frozen schema type differs from the built schema type")
	}
	got, err := ProtocolSchemaJSON(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	want, err := ProtocolSchemaJSON(ctx, WithSchema(r, build))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("GetProviderSchema differs between the frozen and the built schema")
	}
}
