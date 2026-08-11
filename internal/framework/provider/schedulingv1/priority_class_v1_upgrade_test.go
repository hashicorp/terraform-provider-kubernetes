// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	schedulingv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/schedulingv1"
)

// TestUpgradeStateV0toV1 is a unit test that verifies the SDK v2 → framework
// state upgrader logic without any network calls or real Kubernetes cluster.
func TestUpgradeStateV0toV1(t *testing.T) {
	t.Parallel()

	// Build the tftypes value that mirrors SDK v2 JSON state (schema version 0).
	// Metadata was a TypeList — a list with one object.
	v0Type := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"value":             tftypes.Number,
			"description":       tftypes.String,
			"global_default":    tftypes.Bool,
			"preemption_policy": tftypes.String,
			"metadata": tftypes.List{
				ElementType: tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"name":             tftypes.String,
						"generate_name":    tftypes.String,
						"annotations":      tftypes.Map{ElementType: tftypes.String},
						"labels":           tftypes.Map{ElementType: tftypes.String},
						"resource_version": tftypes.String,
						"uid":              tftypes.String,
						"generation":       tftypes.Number,
					},
				},
			},
		},
	}

	v0Value := tftypes.NewValue(v0Type, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "tf-acc-test-pc-upgrade"),
		"value":             tftypes.NewValue(tftypes.Number, 100),
		"description":       tftypes.NewValue(tftypes.String, ""),
		"global_default":    tftypes.NewValue(tftypes.Bool, false),
		"preemption_policy": tftypes.NewValue(tftypes.String, "Never"),
		"metadata": tftypes.NewValue(
			tftypes.List{
				ElementType: tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"name":             tftypes.String,
						"generate_name":    tftypes.String,
						"annotations":      tftypes.Map{ElementType: tftypes.String},
						"labels":           tftypes.Map{ElementType: tftypes.String},
						"resource_version": tftypes.String,
						"uid":              tftypes.String,
						"generation":       tftypes.Number,
					},
				},
			},
			[]tftypes.Value{
				tftypes.NewValue(
					tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"name":             tftypes.String,
							"generate_name":    tftypes.String,
							"annotations":      tftypes.Map{ElementType: tftypes.String},
							"labels":           tftypes.Map{ElementType: tftypes.String},
							"resource_version": tftypes.String,
							"uid":              tftypes.String,
							"generation":       tftypes.Number,
						},
					},
					map[string]tftypes.Value{
						"name":             tftypes.NewValue(tftypes.String, "tf-acc-test-pc-upgrade"),
						"generate_name":    tftypes.NewValue(tftypes.String, ""),
						"annotations":      tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
						"labels":           tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
						"resource_version": tftypes.NewValue(tftypes.String, "1234"),
						"uid":              tftypes.NewValue(tftypes.String, "abc-def-123"),
						"generation":       tftypes.NewValue(tftypes.Number, 0),
					},
				),
			},
		),
	})

	// Build a stub PriorityClassV1 just to get the UpgradeState map.
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("expected upgrader for state version 0")
	}

	rawState := tfsdk.State{
		Schema: *upgrader.PriorSchema,
		Raw:    v0Value,
	}

	req := resource.UpgradeStateRequest{State: &rawState}
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{
			Schema: schedulingv1.PriorityClassV1Schema(),
		},
	}

	upgrader.StateUpgrader(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade produced diagnostics errors: %s", resp.Diagnostics)
	}

	// Verify the upgraded state contains the expected flat metadata
	var upgraded schedulingv1.PriorityClassModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &upgraded)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading upgraded state: %s", resp.Diagnostics)
	}

	if upgraded.ID.ValueString() != "tf-acc-test-pc-upgrade" {
		t.Errorf("id: got %q, want %q", upgraded.ID.ValueString(), "tf-acc-test-pc-upgrade")
	}
	if upgraded.Metadata.Name.ValueString() != "tf-acc-test-pc-upgrade" {
		t.Errorf("metadata.name: got %q, want %q", upgraded.Metadata.Name.ValueString(), "tf-acc-test-pc-upgrade")
	}
	if upgraded.Metadata.ResourceVersion.ValueString() != "1234" {
		t.Errorf("metadata.resource_version: got %q, want %q", upgraded.Metadata.ResourceVersion.ValueString(), "1234")
	}
	if !upgraded.Metadata.GenerateName.IsNull() {
		t.Errorf("metadata.generate_name: expected null, got %q", upgraded.Metadata.GenerateName.ValueString())
	}
	if int(upgraded.Value.ValueInt64()) != 100 {
		t.Errorf("value: got %d, want 100", upgraded.Value.ValueInt64())
	}
}
