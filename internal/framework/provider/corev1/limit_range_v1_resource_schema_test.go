// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func nonNullState() tfsdk.State {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"min": schema.MapAttribute{ElementType: types.StringType, Optional: true},
		},
	}
	raw := tftypes.NewValue(s.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"min": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
	})
	return tfsdk.State{Schema: s, Raw: raw}
}

func nonNullPlan() tfsdk.Plan {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"min": schema.MapAttribute{ElementType: types.StringType, Optional: true},
		},
	}
	raw := tftypes.NewValue(s.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"min": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
	})
	return tfsdk.Plan{Schema: s, Raw: raw}
}

func TestSuppressEmptyMapDiffs(t *testing.T) {
	ctx := context.Background()
	emptyMap, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	nonEmptyMap, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{"cpu": "100m"})
	nullMap := types.MapNull(types.StringType)

	cases := []struct {
		name     string
		config   types.Map
		state    types.Map
		plan     types.Map
		stateRaw tfsdk.State
		planRaw  tfsdk.Plan
		wantPlan types.Map
	}{
		{
			name:     "config is empty map and state is null: retains null state",
			config:   emptyMap,
			state:    nullMap,
			plan:     emptyMap,
			stateRaw: nonNullState(),
			planRaw:  nonNullPlan(),
			wantPlan: nullMap,
		},
		{
			name:     "config is null and state is empty map: retains empty map state",
			config:   nullMap,
			state:    emptyMap,
			plan:     nullMap,
			stateRaw: nonNullState(),
			planRaw:  nonNullPlan(),
			wantPlan: emptyMap,
		},
		{
			name:     "config is empty map and state is empty map: retains empty map",
			config:   emptyMap,
			state:    emptyMap,
			plan:     emptyMap,
			stateRaw: nonNullState(),
			planRaw:  nonNullPlan(),
			wantPlan: emptyMap,
		},
		{
			name:     "config is null and state is null: retains null",
			config:   nullMap,
			state:    nullMap,
			plan:     nullMap,
			stateRaw: nonNullState(),
			planRaw:  nonNullPlan(),
			wantPlan: nullMap,
		},
		{
			name:     "config has values and state is null: preserves planned values",
			config:   nonEmptyMap,
			state:    nullMap,
			plan:     nonEmptyMap,
			stateRaw: nonNullState(),
			planRaw:  nonNullPlan(),
			wantPlan: nonEmptyMap,
		},
		{
			name:     "config is empty map and state has values: preserves planned empty map",
			config:   emptyMap,
			state:    nonEmptyMap,
			plan:     emptyMap,
			stateRaw: nonNullState(),
			planRaw:  nonNullPlan(),
			wantPlan: emptyMap,
		},
		{
			name:     "creating resource (state raw null): does not modify plan",
			config:   emptyMap,
			state:    nullMap,
			plan:     emptyMap,
			stateRaw: tfsdk.State{},
			planRaw:  nonNullPlan(),
			wantPlan: emptyMap,
		},
		{
			name:     "destroying resource (plan raw null): does not modify plan",
			config:   nullMap,
			state:    emptyMap,
			plan:     nullMap,
			stateRaw: nonNullState(),
			planRaw:  tfsdk.Plan{},
			wantPlan: nullMap,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := suppressEmptyMapDiffs()

			req := planmodifier.MapRequest{
				ConfigValue: tc.config,
				StateValue:  tc.state,
				PlanValue:   tc.plan,
				State:       tc.stateRaw,
				Plan:        tc.planRaw,
			}
			resp := &planmodifier.MapResponse{
				PlanValue: tc.plan,
			}

			m.PlanModifyMap(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.wantPlan) {
				t.Errorf("PlanValue = %s, want %s (config=%s, state=%s)",
					resp.PlanValue, tc.wantPlan, tc.config, tc.state)
			}
		})
	}
}
