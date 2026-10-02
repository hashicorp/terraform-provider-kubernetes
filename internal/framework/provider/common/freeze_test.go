// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func freezeTestSchema() schema.Schema {
	leaf := schema.NestedBlockObject{Attributes: map[string]schema.Attribute{"v": schema.StringAttribute{Optional: true}}}
	shared := schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"n": schema.Int64Attribute{Optional: true}},
		Blocks:     map[string]schema.Block{"leaf": schema.ListNestedBlock{NestedObject: leaf}},
	}
	return schema.Schema{
		Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}},
		Blocks: map[string]schema.Block{
			"list": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
				// The same object (and so the same maps) under two parents.
				"a": schema.ListNestedBlock{NestedObject: shared},
				"b": schema.SetNestedBlock{NestedObject: shared},
			}}},
			"single":   schema.SingleNestedBlock{Blocks: map[string]schema.Block{"leaf": schema.ListNestedBlock{NestedObject: leaf}}},
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{Create: true}),
		},
	}
}

func TestFreezeSchemaPreservesTypes(t *testing.T) {
	ctx := context.Background()
	built := freezeTestSchema()
	frozen := FreezeSchema(built)
	if !built.Type().Equal(frozen.Type()) {
		t.Fatalf("type changed:\nbefore %s\nafter  %s", built.Type(), frozen.Type())
	}
	if !built.Type().TerraformType(ctx).Equal(frozen.Type().TerraformType(ctx)) {
		t.Fatal("terraform type changed")
	}
	// The input is not mutated.
	if built.Blocks["list"].(schema.ListNestedBlock).CustomType != nil {
		t.Fatal("FreezeSchema mutated its input")
	}
	list := frozen.Blocks["list"].(schema.ListNestedBlock)
	if list.CustomType == nil || list.NestedObject.CustomType == nil {
		t.Fatal("list block not frozen")
	}
	set := list.NestedObject.Blocks["b"].(schema.SetNestedBlock)
	if set.CustomType == nil || set.NestedObject.CustomType == nil {
		t.Fatal("nested set block not frozen")
	}
	if set.NestedObject.Blocks["leaf"].(schema.ListNestedBlock).CustomType == nil {
		t.Fatal("deep block not frozen")
	}
	if frozen.Blocks["single"].(schema.SingleNestedBlock).CustomType == nil {
		t.Fatal("single block not frozen")
	}
	// An existing CustomType is kept, not replaced by the derived ObjectType.
	if _, ok := frozen.Blocks["timeouts"].(schema.SingleNestedBlock).CustomType.(timeouts.Type); !ok {
		t.Fatal("timeouts CustomType was replaced")
	}
	if _, ok := list.CustomType.(types.ListType); !ok {
		t.Fatalf("unexpected CustomType %T", list.CustomType)
	}
	if diags := frozen.ValidateImplementation(ctx); diags.HasError() {
		t.Fatal(diags)
	}
}

func TestFrozenSchemaBuildsOnce(t *testing.T) {
	calls := 0
	get := FrozenSchema(func(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
		calls++
		resp.Schema = freezeTestSchema()
	})
	for i := 0; i < 3; i++ {
		var resp resource.SchemaResponse
		get(context.Background(), resource.SchemaRequest{}, &resp)
		if resp.Schema.Blocks["list"].(schema.ListNestedBlock).CustomType == nil {
			t.Fatal("schema not frozen")
		}
	}
	if calls != 1 {
		t.Fatalf("builder ran %d times, want 1", calls)
	}
}
