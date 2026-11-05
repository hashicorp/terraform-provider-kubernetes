// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestQuantityValueSemanticEquals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		prior    QuantityValue
		proposed QuantityValue
		want     bool
	}{
		// The cases that failed live with "Provider produced inconsistent result after apply".
		{"1000m equals 1", NewQuantityValue("1000m"), NewQuantityValue("1"), true},
		{"2048Mi equals 2Gi", NewQuantityValue("2048Mi"), NewQuantityValue("2Gi"), true},
		{"0.5 equals 500m", NewQuantityValue("0.5"), NewQuantityValue("500m"), true},
		{"2000m equals 2", NewQuantityValue("2000m"), NewQuantityValue("2"), true},
		{"identical", NewQuantityValue("4"), NewQuantityValue("4"), true},
		{"different value", NewQuantityValue("1"), NewQuantityValue("2"), false},
		{"1Gi is not 1G", NewQuantityValue("1Gi"), NewQuantityValue("1G"), false},
		{"unparseable prior is never equivalent", NewQuantityValue("abc"), NewQuantityValue("abc"), false},
		{"unparseable proposed is never equivalent", NewQuantityValue("1"), NewQuantityValue("abc"), false},
		{"null prior", QuantityValue{StringValue: types.StringNull()}, NewQuantityValue("1"), false},
		{"null proposed", NewQuantityValue("1"), QuantityValue{StringValue: types.StringNull()}, false},
		{"unknown prior", QuantityValue{StringValue: types.StringUnknown()}, NewQuantityValue("1"), false},
		{"unknown proposed", NewQuantityValue("1"), QuantityValue{StringValue: types.StringUnknown()}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, diags := tc.prior.StringSemanticEquals(context.Background(), tc.proposed)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQuantityValueSemanticEqualsWrongType(t *testing.T) {
	t.Parallel()

	_, diags := NewQuantityValue("1").StringSemanticEquals(context.Background(), types.StringValue("1"))
	if !diags.HasError() {
		t.Fatal("expected an error when compared with a plain string value")
	}
}

// The type must round-trip through the framework's conversion paths, or a map of
// QuantityType fails at decode time rather than in a unit test.
func TestQuantityTypeRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	typ := QuantityType{}

	if !typ.Equal(QuantityType{}) {
		t.Error("QuantityType should equal itself")
	}
	if typ.Equal(types.StringType) {
		t.Error("QuantityType must not equal the plain string type")
	}
	if !typ.ValueType(ctx).Type(ctx).Equal(typ) {
		t.Error("ValueType().Type() should return QuantityType")
	}

	for _, in := range []basetypes.StringValue{types.StringValue("2Gi"), types.StringNull(), types.StringUnknown()} {
		v, diags := typ.ValueFromString(ctx, in)
		if diags.HasError() {
			t.Fatalf("ValueFromString(%s): %v", in, diags)
		}
		tfVal, err := v.ToTerraformValue(ctx)
		if err != nil {
			t.Fatalf("ToTerraformValue(%s): %v", in, err)
		}
		back, err := typ.ValueFromTerraform(ctx, tfVal)
		if err != nil {
			t.Fatalf("ValueFromTerraform(%s): %v", in, err)
		}
		if !back.Equal(v) {
			t.Errorf("round trip changed %s into %s", v, back)
		}
	}

	// A map with this element type must build and report the element type back.
	m, diags := types.MapValue(typ, map[string]attr.Value{"cpu": NewQuantityValue("500m")})
	if diags.HasError() {
		t.Fatalf("MapValue: %v", diags)
	}
	if !m.ElementType(ctx).Equal(typ) {
		t.Errorf("element type = %s, want QuantityType", m.ElementType(ctx))
	}
}

func TestQuantityMapValidator(t *testing.T) {
	t.Parallel()

	mapOf := func(elems map[string]attr.Value) types.Map {
		return types.MapValueMust(QuantityType{}, elems)
	}

	cases := []struct {
		name       string
		value      types.Map
		wantErrors int
	}{
		{"null map is not validated", types.MapNull(QuantityType{}), 0},
		{"unknown map is not validated", types.MapUnknown(QuantityType{}), 0},
		{"empty map", mapOf(map[string]attr.Value{}), 0},
		{"valid quantities", mapOf(map[string]attr.Value{
			"limits.cpu":    NewQuantityValue("2"),
			"limits.memory": NewQuantityValue("2048Mi"),
			"pods":          NewQuantityValue("4"),
			"requests.cpu":  NewQuantityValue("500m"),
		}), 0},
		{"unknown element is deferred to apply", mapOf(map[string]attr.Value{
			"pods": QuantityValue{StringValue: types.StringUnknown()},
		}), 0},
		{"invalid quantity is rejected", mapOf(map[string]attr.Value{"pods": NewQuantityValue("many")}), 1},
		{"empty string is rejected", mapOf(map[string]attr.Value{"pods": NewQuantityValue("")}), 1},
		{"null element is rejected", mapOf(map[string]attr.Value{"pods": QuantityValue{StringValue: types.StringNull()}}), 1},
		{"every bad element is reported", mapOf(map[string]attr.Value{
			"a": NewQuantityValue("x"),
			"b": NewQuantityValue("y"),
			"c": NewQuantityValue("1"),
		}), 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := validator.MapRequest{Path: path.Root("hard"), ConfigValue: tc.value}
			resp := &validator.MapResponse{}
			QuantityMapValidator().ValidateMap(context.Background(), req, resp)

			if got := resp.Diagnostics.ErrorsCount(); got != tc.wantErrors {
				t.Errorf("got %d errors, want %d: %v", got, tc.wantErrors, resp.Diagnostics)
			}
		})
	}
}
