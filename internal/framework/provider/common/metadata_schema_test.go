// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestRequiresReplaceUnlessSDKv2Unset covers the exemption that keeps a plan run without
// refresh from destroying objects whose state SDKv2 wrote.
//
// This cannot be an acceptance test: terraform-plugin-testing has no way to skip the refresh
// before a plan (only RefreshState, which is refresh-only), so the harness always normalises
// "" away through Read before any plan check runs.
func TestRequiresReplaceUnlessSDKv2Unset(t *testing.T) {
	cases := []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{
			// The bug: SDKv2 stores "" for an unset generate_name, the config plans null.
			name:  "SDKv2 unset against planned null does not replace",
			state: types.StringValue(""),
			plan:  types.StringNull(),
			want:  false,
		},
		{
			name:  "state null against planned null does not replace",
			state: types.StringNull(),
			plan:  types.StringNull(),
			want:  false,
		},
		{
			name:  "unchanged value does not replace",
			state: types.StringValue("demo-"),
			plan:  types.StringValue("demo-"),
			want:  false,
		},
		{
			name:  "changed value still replaces",
			state: types.StringValue("demo-"),
			plan:  types.StringValue("other-"),
			want:  true,
		},
		{
			// Only the "" -> null direction is exempt. Adding generate_name to an object
			// that never had one is a real change.
			name:  "SDKv2 unset against a real value still replaces",
			state: types.StringValue(""),
			plan:  types.StringValue("demo-"),
			want:  true,
		},
		{
			name:  "removing a real value still replaces",
			state: types.StringValue("demo-"),
			plan:  types.StringNull(),
			want:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := generateNamePlanModifier(t)

			req := planmodifier.StringRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
				// RequiresReplaceIf returns early unless both are set: a null state means
				// create, a null plan means destroy, and neither replaces anything.
				State: nonNullState(),
				Plan:  nonNullPlan(),
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}

			m.PlanModifyString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
			}
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %t, want %t (state %s, plan %s)",
					resp.RequiresReplace, tc.want, tc.state, tc.plan)
			}
		})
	}
}

// Guards the assumption the exemption rests on: the plain RequiresReplace would have
// replaced in the "" -> null case. If the framework ever stopped treating "" and null as
// different, the exemption would be dead code rather than a fix.
func TestRequiresReplaceReplacesOnSDKv2Unset(t *testing.T) {
	m, ok := stringplanmodifier.RequiresReplace().(interface {
		PlanModifyString(context.Context, planmodifier.StringRequest, *planmodifier.StringResponse)
	})
	if !ok {
		t.Fatal("plan modifier does not implement PlanModifyString")
	}

	req := planmodifier.StringRequest{
		StateValue: types.StringValue(""),
		PlanValue:  types.StringNull(),
		State:      nonNullState(),
		Plan:       nonNullPlan(),
	}
	resp := &planmodifier.StringResponse{PlanValue: types.StringNull()}

	m.PlanModifyString(context.Background(), req, resp)

	if !resp.RequiresReplace {
		t.Error("RequiresReplace = false for \"\" -> null; the exemption is no longer needed")
	}
}

// RequiresReplaceIf returns early when either raw value is null — that is create or destroy,
// where nothing is replaced. These give it a non-null state and plan so the comparison runs.
func probeSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"generate_name": schema.StringAttribute{Optional: true},
		},
	}
}

func probeRaw() tftypes.Value {
	s := probeSchema()
	return tftypes.NewValue(s.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"generate_name": tftypes.NewValue(tftypes.String, nil),
	})
}

func nonNullState() tfsdk.State {
	return tfsdk.State{Schema: probeSchema(), Raw: probeRaw()}
}

func nonNullPlan() tfsdk.Plan {
	return tfsdk.Plan{Schema: probeSchema(), Raw: probeRaw()}
}

// generateNamePlanModifier returns the plan modifier MetadataSchema actually wires to
// generate_name, so this tests the schema rather than a copy of the logic.
func generateNamePlanModifier(t *testing.T) planmodifier.String {
	t.Helper()

	block := MetadataSchema("namespace", true)
	attr, ok := block.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("generate_name is not a StringAttribute")
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatalf("generate_name has %d plan modifiers, want 1", len(attr.PlanModifiers))
	}
	return attr.PlanModifiers[0]
}
