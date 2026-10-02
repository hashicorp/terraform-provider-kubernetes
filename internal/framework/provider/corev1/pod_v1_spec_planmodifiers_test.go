// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPodV1SpecComputedStringReplacement(t *testing.T) {
	ctx := context.Background()
	old := types.StringValue("api-value")
	raw, err := old.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name                   string
		state, config, planned types.String
		want                   types.String
		replace                bool
	}{
		{"API default during unrelated update", old, types.StringNull(), types.StringUnknown(), old, false},
		{"configured unknown remains unknown", old, types.StringUnknown(), types.StringUnknown(), types.StringUnknown(), true},
		{"configured edit", old, types.StringValue("new"), types.StringValue("new"), types.StringValue("new"), true},
		{"removed containing block", old, types.StringNull(), types.StringNull(), types.StringNull(), true},
		{"added empty default", types.StringNull(), types.StringNull(), types.StringValue(""), types.StringValue(""), false},
		{"added nonzero configured value", types.StringNull(), types.StringValue("new"), types.StringValue("new"), types.StringValue("new"), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State: tfsdk.State{Raw: raw}, Plan: tfsdk.Plan{Raw: raw},
				StateValue: test.state, ConfigValue: test.config, PlanValue: test.planned,
			}
			resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
			for _, modifier := range podString(false, true, true, "").PlanModifiers {
				modifier.PlanModifyString(ctx, req, &resp)
				req.PlanValue = resp.PlanValue
			}
			if resp.Diagnostics.HasError() || !resp.PlanValue.Equal(test.want) || resp.RequiresReplace != test.replace {
				t.Fatalf("value=%s replace=%t diagnostics=%v; want value=%s replace=%t",
					resp.PlanValue, resp.RequiresReplace, resp.Diagnostics, test.want, test.replace)
			}
		})
	}
}

func TestPodV1SpecScalarZeroReplacement(t *testing.T) {
	ctx := context.Background()
	raw, err := types.StringValue("existing-resource").ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name           string
		old, plan      types.Bool
		requiresChange bool
	}{
		{"new false default", types.BoolNull(), types.BoolValue(false), false},
		{"new nonzero value", types.BoolNull(), types.BoolValue(true), true},
		{"reset known value", types.BoolValue(true), types.BoolValue(false), true},
		{"remove known value", types.BoolValue(true), types.BoolNull(), true},
	} {
		t.Run("bool/"+test.name, func(t *testing.T) {
			req := planmodifier.BoolRequest{
				State: tfsdk.State{Raw: raw}, Plan: tfsdk.Plan{Raw: raw},
				StateValue: test.old, PlanValue: test.plan, ConfigValue: test.plan,
			}
			resp := planmodifier.BoolResponse{PlanValue: test.plan}
			for _, modifier := range podBool(false, true, false).PlanModifiers {
				modifier.PlanModifyBool(ctx, req, &resp)
				req.PlanValue = resp.PlanValue
			}
			if resp.Diagnostics.HasError() || !resp.PlanValue.Equal(test.plan) || resp.RequiresReplace != test.requiresChange {
				t.Fatalf("value=%s replace=%t diagnostics=%v", resp.PlanValue, resp.RequiresReplace, resp.Diagnostics)
			}
		})
	}
	for _, test := range []struct {
		name           string
		old, plan      types.Int64
		requiresChange bool
	}{
		{"new zero default", types.Int64Null(), types.Int64Value(0), false},
		{"new nonzero value", types.Int64Null(), types.Int64Value(1), true},
		{"reset known value", types.Int64Value(1), types.Int64Value(0), true},
		{"remove known value", types.Int64Value(1), types.Int64Null(), true},
	} {
		t.Run("int/"+test.name, func(t *testing.T) {
			req := planmodifier.Int64Request{
				State: tfsdk.State{Raw: raw}, Plan: tfsdk.Plan{Raw: raw},
				StateValue: test.old, PlanValue: test.plan, ConfigValue: test.plan,
			}
			resp := planmodifier.Int64Response{PlanValue: test.plan}
			for _, modifier := range podInt(false, false, true, 0).PlanModifiers {
				modifier.PlanModifyInt64(ctx, req, &resp)
				req.PlanValue = resp.PlanValue
			}
			if resp.Diagnostics.HasError() || !resp.PlanValue.Equal(test.plan) || resp.RequiresReplace != test.requiresChange {
				t.Fatalf("value=%s replace=%t diagnostics=%v", resp.PlanValue, resp.RequiresReplace, resp.Diagnostics)
			}
		})
	}
}

func TestPodV1SpecComputedReferenceReplacement(t *testing.T) {
	ctx := context.Background()
	for _, child := range []string{"name", "condition_type"} {
		schema := podComputedReferences(child)
		value := func(name string) types.List {
			object := types.ObjectValueMust(map[string]attr.Type{child: types.StringType},
				map[string]attr.Value{child: types.StringValue(name)})
			return types.ListValueMust(schema.ElementType, []attr.Value{object})
		}
		old, changed := value("api-value"), value("configured-new")
		raw, err := old.ToTerraformValue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		null, unknown := types.ListNull(schema.ElementType), types.ListUnknown(schema.ElementType)
		for _, test := range []struct {
			name            string
			config, planned types.List
			want            types.List
			replace         bool
		}{
			{"API default during unrelated update", null, unknown, old, false},
			{"configured unknown remains unknown", unknown, unknown, unknown, true},
			{"configured edit", changed, changed, changed, true},
			{"removed containing block", null, null, null, true},
		} {
			t.Run(child+"/"+test.name, func(t *testing.T) {
				req := planmodifier.ListRequest{
					State: tfsdk.State{Raw: raw}, Plan: tfsdk.Plan{Raw: raw},
					StateValue: old, ConfigValue: test.config, PlanValue: test.planned,
				}
				resp := planmodifier.ListResponse{PlanValue: req.PlanValue}
				for _, modifier := range schema.PlanModifiers {
					modifier.PlanModifyList(ctx, req, &resp)
					req.PlanValue = resp.PlanValue
				}
				if resp.Diagnostics.HasError() || !resp.PlanValue.Equal(test.want) || resp.RequiresReplace != test.replace {
					t.Fatalf("value=%s replace=%t diagnostics=%v; want value=%s replace=%t",
						resp.PlanValue, resp.RequiresReplace, resp.Diagnostics, test.want, test.replace)
				}
			})
		}
	}
}
