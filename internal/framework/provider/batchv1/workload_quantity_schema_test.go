// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestWorkloadQuantitySemanticEquality(t *testing.T) {
	for _, tc := range []struct {
		name, left, right string
		equal             bool
	}{
		{"cpu", "1", "1000m", true},
		{"memory", "1Gi", "1024Mi", true},
		{"decimal", "0.1", "100m", true},
		{"different", "1Gi", "2Gi", false},
		{"invalid-left", "invalid", "1", false},
		{"invalid-right", "1", "invalid", false},
		{"both-invalid", "invalid", "invalid", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := workloadQuantityValue{StringValue: types.StringValue(tc.left)}
			right := workloadQuantityValue{StringValue: types.StringValue(tc.right)}
			got, diags := left.StringSemanticEquals(context.Background(), right)
			if diags.HasError() || got != tc.equal {
				t.Fatalf("equality=%t, want %t; diagnostics=%s", got, tc.equal, diags)
			}
			// Structural equality must remain exact; only semantic equality suppresses normalization.
			if got := left.Equal(right); got != (tc.left == tc.right) {
				t.Fatalf("structural equality=%t for %q and %q", got, tc.left, tc.right)
			}
		})
	}
	for _, value := range []basetypes.StringValue{types.StringNull(), types.StringUnknown()} {
		quantity := workloadQuantityValue{StringValue: value}
		if equal, _ := quantity.StringSemanticEquals(context.Background(), quantity); equal {
			t.Fatal("null and unknown values must not participate in semantic equality")
		}
	}
}

func TestWorkloadQuantityTypeConversion(t *testing.T) {
	ctx := context.Background()
	typ := workloadQuantityType{}
	for _, raw := range []tftypes.Value{
		tftypes.NewValue(tftypes.String, nil),
		tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		tftypes.NewValue(tftypes.String, "1024Mi"),
	} {
		value, err := typ.ValueFromTerraform(ctx, raw)
		if err != nil {
			t.Fatal(err)
		}

		if !value.Type(ctx).Equal(typ) {
			t.Fatalf("wrong value type: %T", value)
		}
		roundTrip, err := value.ToTerraformValue(ctx)
		if err != nil || !roundTrip.Equal(raw) {
			t.Fatalf("roundtrip: got %s, want %s; error=%v", roundTrip, raw, err)
		}
	}
	if _, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.Number, 1)); err == nil {
		t.Fatal("expected error converting a number to quantity")
	}
	value, diags := typ.ValueFromString(ctx, types.StringValue("1Gi"))
	if diags.HasError() || !value.Type(ctx).Equal(typ) {
		t.Fatalf("ValueFromString failed: %s", diags)
	}
	if _, ok := typ.ValueType(ctx).(basetypes.StringValuableWithSemanticEquals); !ok {
		t.Fatal("quantity values lost semantic equality")
	}
	values := types.MapValueMust(typ, map[string]attr.Value{"memory": value})
	raw, err := values.ToTerraformValue(ctx)
	if err != nil || !raw.Type().Equal(tftypes.Map{ElementType: tftypes.String}) {
		t.Fatalf("quantity map persisted type changed: %s; error=%v", raw, err)
	}
}

func TestWorkloadQuantityFormattingDoesNotReplace(t *testing.T) {
	ctx := context.Background()
	quantityMap := func(cpu string) types.Map {
		return types.MapValueMust(workloadQuantityType{}, map[string]attr.Value{
			"cpu": workloadQuantityValue{StringValue: types.StringValue(cpu)},
		})
	}
	field := workloadResourcesObject(false).Attributes["limits"].(schema.MapAttribute)
	request := planmodifier.MapRequest{
		State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
		Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
		StateValue:  quantityMap("1000m"),
		PlanValue:   quantityMap("1"),
		ConfigValue: quantityMap("1"),
	}
	response := planmodifier.MapResponse{PlanValue: request.PlanValue}
	for _, modifier := range field.PlanModifiers {
		modifier.PlanModifyMap(ctx, request, &response)
		request.PlanValue = response.PlanValue
	}
	if response.Diagnostics.HasError() || response.RequiresReplace || !response.PlanValue.Equal(request.StateValue) {
		t.Fatalf("equivalent quantities must retain state without replacement: %#v", response)
	}
}

func TestWorkloadListReplacement(t *testing.T) {
	ctx := context.Background()
	objType := types.ObjectType{AttrTypes: map[string]attr.Type{"computed": types.StringType}}
	list := func(value types.String) types.List {
		return types.ListValueMust(objType, []attr.Value{
			types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{"computed": value}),
		})
	}
	for _, tc := range []struct {
		name          string
		plan, config  types.String
		shouldReplace bool
	}{
		{"computed-unknown", types.StringUnknown(), types.StringNull(), false},
		{"configured-unknown", types.StringUnknown(), types.StringUnknown(), true},
		{"unchanged", types.StringValue("assigned"), types.StringValue("assigned"), false},
		{"changed", types.StringValue("new"), types.StringValue("new"), true},
		{"removed", types.StringNull(), types.StringNull(), true},
		{"explicit-empty", types.StringValue(""), types.StringValue(""), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, updatable := range []bool{false, true} {
				request := planmodifier.ListRequest{
					State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
					Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
					StateValue:  list(types.StringValue("assigned")),
					PlanValue:   list(tc.plan),
					ConfigValue: list(tc.config),
				}
				response := planmodifier.ListResponse{PlanValue: request.PlanValue}
				for _, modifier := range []planmodifier.List{workloadListRequiresReplace()} {
					modifier.PlanModifyList(ctx, request, &response)
				}
				if response.Diagnostics.HasError() || response.RequiresReplace != tc.shouldReplace {
					t.Fatalf("updatable=%t: replacement=%t, want %t; diagnostics=%s",
						updatable, response.RequiresReplace, tc.shouldReplace, response.Diagnostics)
				}
			}
		})
	}
}

func TestWorkloadQuantityAncestorReplacement(t *testing.T) {
	ctx := context.Background()
	objType := types.ObjectType{AttrTypes: map[string]attr.Type{"quantity": workloadQuantityType{}}}
	list := func(value string) types.List {
		return types.ListValueMust(objType, []attr.Value{
			types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
				"quantity": workloadQuantityValue{StringValue: types.StringValue(value)},
			}),
		})
	}
	for _, tc := range []struct {
		quantity string
		replace  bool
	}{
		{"1", false},
		{"2", true},
		{"invalid", true},
	} {
		request := planmodifier.ListRequest{
			State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
			StateValue:  list("1000m"),
			PlanValue:   list(tc.quantity),
			ConfigValue: list(tc.quantity),
		}
		response := planmodifier.ListResponse{PlanValue: request.PlanValue}
		for _, modifier := range []planmodifier.List{workloadListRequiresReplace()} {
			modifier.PlanModifyList(ctx, request, &response)
		}
		if response.Diagnostics.HasError() || response.RequiresReplace != tc.replace {
			t.Fatalf("quantity=%q: replacement=%t, want %t", tc.quantity, response.RequiresReplace, tc.replace)
		}
	}
}
