// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schematest

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func TestUnfrozenBlockPaths(t *testing.T) {
	leaf := schema.NestedBlockObject{Attributes: map[string]schema.Attribute{"v": schema.StringAttribute{Optional: true}}}
	s := schema.Schema{Blocks: map[string]schema.Block{
		"list": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
			"set": schema.SetNestedBlock{NestedObject: leaf},
		}}},
		"single": schema.SingleNestedBlock{Blocks: map[string]schema.Block{"leaf": schema.ListNestedBlock{NestedObject: leaf}}},
	}}
	want := []string{"list", "list.set", "single", "single.leaf"}
	if got := UnfrozenBlockPaths(s.Blocks); !reflect.DeepEqual(got, want) {
		t.Fatalf("unfrozen schema: got %v, want %v", got, want)
	}
	if got := UnfrozenBlockPaths(common.FreezeSchema(s).Blocks); len(got) != 0 {
		t.Fatalf("frozen schema still reports %v", got)
	}
}
