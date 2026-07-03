// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubemanifest

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAttrToGo_scalars(t *testing.T) {
	ctx := context.Background()
	if g, _ := AttrToGo(ctx, types.StringValue("x")); g != "x" {
		t.Errorf("string: got %v", g)
	}
	if g, _ := AttrToGo(ctx, types.BoolValue(true)); g != true {
		t.Errorf("bool: got %v", g)
	}
	if g, _ := AttrToGo(ctx, types.StringNull()); g != nil {
		t.Errorf("null should be nil, got %v", g)
	}
	// integral number → int64
	if g, _ := AttrToGo(ctx, types.NumberValue(big.NewFloat(3))); g != int64(3) {
		t.Errorf("number 3: got %v (%T)", g, g)
	}
}

func TestAttrToGo_unknownIsError(t *testing.T) {
	if _, err := AttrToGo(context.Background(), types.StringUnknown()); err == nil {
		t.Fatal("unknown value should error")
	}
}

// TestAttrToGo_nullLeafPreserved is THE critical test: a null leaf must serialize to
// JSON null (SSA field removal), not be dropped.
func TestAttrToGo_nullLeafPreserved(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType, "gone": types.StringType},
		map[string]attr.Value{"a": types.StringValue("1"), "gone": types.StringNull()},
	)
	got, err := AttrToGo(context.Background(), obj)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	b, _ := json.Marshal(got)
	if string(b) != `{"a":"1","gone":null}` {
		t.Fatalf("null leaf must be preserved as JSON null, got %s", string(b))
	}
}

func TestAttrToGo_nestedAndDynamic(t *testing.T) {
	inner := types.ObjectValueMust(
		map[string]attr.Type{"replicas": types.NumberType},
		map[string]attr.Value{"replicas": types.NumberValue(big.NewFloat(3))},
	)
	spec := types.ObjectValueMust(
		map[string]attr.Type{"spec": inner.Type(context.Background())},
		map[string]attr.Value{"spec": inner},
	)
	dyn := types.DynamicValue(spec)
	got, err := AttrToGo(context.Background(), dyn)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	b, _ := json.Marshal(got)
	if string(b) != `{"spec":{"replicas":3}}` {
		t.Fatalf("nested/dynamic conversion wrong, got %s", string(b))
	}
}

func TestPruneNulls(t *testing.T) {
	m := map[string]interface{}{
		"keep":   "1",
		"gone":   nil,
		"nested": map[string]interface{}{"a": int64(2), "drop": nil},
		"list":   []interface{}{map[string]interface{}{"x": "y", "z": nil}},
	}
	PruneNulls(m)
	if _, ok := m["gone"]; ok {
		t.Error("top-level null must be pruned")
	}
	n := m["nested"].(map[string]interface{})
	if _, ok := n["drop"]; ok {
		t.Error("nested null must be pruned")
	}
	if n["a"] != int64(2) {
		t.Error("non-null nested value must remain")
	}
	le := m["list"].([]interface{})[0].(map[string]interface{})
	if _, ok := le["z"]; ok {
		t.Error("null inside list element must be pruned")
	}
	if le["x"] != "y" {
		t.Error("non-null in list element must remain")
	}
}

func TestDeepMerge(t *testing.T) {
	dst := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "app", "namespace": "default"},
	}
	src := map[string]interface{}{
		"metadata": map[string]interface{}{"annotations": map[string]interface{}{"k": "v"}},
		"spec":     map[string]interface{}{"replicas": int64(3)},
	}
	out := DeepMerge(dst, src)
	md := out["metadata"].(map[string]interface{})
	if md["name"] != "app" || md["namespace"] != "default" {
		t.Fatalf("identity fields must survive merge: %v", md)
	}
	if _, ok := md["annotations"]; !ok {
		t.Fatalf("annotations must be merged in: %v", md)
	}
	if out["spec"].(map[string]interface{})["replicas"] != int64(3) {
		t.Fatalf("spec.replicas must be merged: %v", out["spec"])
	}
}
