// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// apiTerraformValue builds the Job and CronJob specs as tftypes values and
// apiValue converts them to attr.Value once, at the root. That equals
// converting at every level only while each type in the spec returns
// ValueFromTerraform input unchanged from ToTerraformValue. These tests fail
// when a type that could break this (for example a CustomType that normalizes
// values) is added to the spec, so the shortcut must be revisited.
func TestBatchSpecTypesRoundTripTerraformValues(t *testing.T) {
	ctx := context.Background()
	for name, field := range map[string]valueField{
		"job":      jobSpecValueField(),
		"cron_job": cronJobSpecValueField(),
	} {
		t.Run(name, func(t *testing.T) {
			leaves := map[string]attr.Type{}
			checkRoundTripTypes(t, "spec", field.typ, leaves)
			for key, typ := range leaves {
				for _, in := range sampleLeafValues(ctx, typ) {
					assertRoundTrip(ctx, t, key, typ, in)
				}
			}
			assertRoundTrip(ctx, t, "spec", field.typ, sampleTerraformValue(ctx, field.typ))
			assertRoundTrip(ctx, t, "spec", field.typ, tftypes.NewValue(field.typ.TerraformType(ctx), nil))
		})
	}
}

// checkRoundTripTypes fails for any type in the tree other than the basetypes
// collections, objects and primitives and the string-backed quantity type, and
// collects one representative per leaf type.
func checkRoundTripTypes(t *testing.T, path string, typ attr.Type, leaves map[string]attr.Type) {
	t.Helper()
	switch v := typ.(type) {
	case basetypes.ListType:
		checkRoundTripTypes(t, path+"[]", v.ElemType, leaves)
	case basetypes.SetType:
		checkRoundTripTypes(t, path+"[]", v.ElemType, leaves)
	case basetypes.MapType:
		checkRoundTripTypes(t, path+"{}", v.ElemType, leaves)
	case basetypes.ObjectType:
		for name, child := range v.AttrTypes {
			checkRoundTripTypes(t, path+"."+name, child, leaves)
		}
	case basetypes.StringType, basetypes.BoolType, basetypes.Int64Type, basetypes.Int32Type,
		basetypes.Float64Type, basetypes.NumberType, workloadQuantityType:
		leaves[fmt.Sprintf("%T", typ)] = typ
	default:
		t.Errorf("%s: type %T (%s) is not known to round-trip through tftypes; check apiTerraformValue", path, typ, typ)
	}
}

func sampleLeafValues(ctx context.Context, typ attr.Type) []tftypes.Value {
	tf := typ.TerraformType(ctx)
	values := []tftypes.Value{tftypes.NewValue(tf, nil), tftypes.NewValue(tf, tftypes.UnknownValue)}
	switch {
	case tf.Is(tftypes.String):
		values = append(values, tftypes.NewValue(tf, ""), tftypes.NewValue(tf, "500m"), tftypes.NewValue(tf, "1Gi"))
	case tf.Is(tftypes.Number):
		values = append(values, tftypes.NewValue(tf, big.NewFloat(0)), tftypes.NewValue(tf, big.NewFloat(42)))
	case tf.Is(tftypes.Bool):
		values = append(values, tftypes.NewValue(tf, false), tftypes.NewValue(tf, true))
	}
	return values
}

// sampleTerraformValue returns a known value of typ with one element in every
// collection and every object attribute set.
func sampleTerraformValue(ctx context.Context, typ attr.Type) tftypes.Value {
	tf := typ.TerraformType(ctx)
	switch v := typ.(type) {
	case basetypes.ListType:
		return tftypes.NewValue(tf, []tftypes.Value{sampleTerraformValue(ctx, v.ElemType)})
	case basetypes.SetType:
		return tftypes.NewValue(tf, []tftypes.Value{sampleTerraformValue(ctx, v.ElemType)})
	case basetypes.MapType:
		return tftypes.NewValue(tf, map[string]tftypes.Value{"key": sampleTerraformValue(ctx, v.ElemType)})
	case basetypes.ObjectType:
		attrs := make(map[string]tftypes.Value, len(v.AttrTypes))
		for name, child := range v.AttrTypes {
			attrs[name] = sampleTerraformValue(ctx, child)
		}
		return tftypes.NewValue(tf, attrs)
	}
	samples := sampleLeafValues(ctx, typ)
	return samples[len(samples)-1]
}

func assertRoundTrip(ctx context.Context, t *testing.T, path string, typ attr.Type, in tftypes.Value) {
	t.Helper()
	value, err := typ.ValueFromTerraform(ctx, in)
	if err != nil {
		t.Fatalf("%s: ValueFromTerraform: %v", path, err)
	}
	out, err := value.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("%s: ToTerraformValue: %v", path, err)
	}
	if !out.Equal(in) {
		t.Errorf("%s: %s does not round-trip: %s became %s", path, typ, in, out)
	}
}
