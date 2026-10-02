// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestWorkloadNoopComputedPlan(t *testing.T) {
	ctx := context.Background()
	testSchema := schema.Schema{Attributes: map[string]schema.Attribute{
		"configured": schema.StringAttribute{Optional: true},
		"computed":   schema.StringAttribute{Computed: true},
	}}
	raw := func(configured, computed any) tftypes.Value {
		return tftypes.NewValue(testSchema.Type().TerraformType(ctx), map[string]tftypes.Value{
			"configured": tftypes.NewValue(tftypes.String, configured),
			"computed":   tftypes.NewValue(tftypes.String, computed),
		})
	}
	for _, tc := range []struct {
		name                string
		prior, plan, config any
		restore             bool
	}{
		{"known-values-unchanged", "unchanged", "unchanged", "unchanged", true},
		{"real-update", "before", "after", "after", false},
		{"legacy-normalization", "", nil, nil, false},
		{"explicit-empty-added", nil, "", "", false},
		{"unknown-configuration", "before", tftypes.UnknownValue, tftypes.UnknownValue, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: testSchema, Raw: raw(tc.config, nil)},
				Plan:   tfsdk.Plan{Schema: testSchema, Raw: raw(tc.plan, tftypes.UnknownValue)},
				State:  tfsdk.State{Schema: testSchema, Raw: raw(tc.prior, "server-value")},
			}
			response := resource.ModifyPlanResponse{Plan: request.Plan}
			preserveWorkloadNoopPlan(ctx, request, &response)
			if got := response.Plan.Raw.Equal(request.State.Raw); got != tc.restore {
				t.Fatalf("restored state=%t, want %t; plan=%s", got, tc.restore, response.Plan.Raw)
			}
			if !tc.restore && !response.Plan.Raw.Equal(request.Plan.Raw) {
				t.Fatal("non-noop plan was modified")
			}
		})
	}
}
