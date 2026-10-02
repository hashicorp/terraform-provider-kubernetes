// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	appsv1 "k8s.io/api/apps/v1"
)

func TestExpandDaemonSetStrategy_DefaultWhenOmittedOrUnknown(t *testing.T) {
	t.Parallel()

	for _, value := range []types.List{
		types.ListNull(daemonSetStrategyObjectType()),
		types.ListUnknown(daemonSetStrategyObjectType()),
	} {
		strategy, diags := expandDaemonSetStrategyModel(context.Background(), value, path.Root("strategy"))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if strategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
			t.Fatalf("expected strategy type %q, got %q", appsv1.RollingUpdateDaemonSetStrategyType, strategy.Type)
		}
		if strategy.RollingUpdate != nil {
			t.Fatalf("expected nil rolling_update when strategy omitted, got %#v", strategy.RollingUpdate)
		}
	}
}

func TestExpandDaemonSetStrategy_OnDeleteClearsRollingUpdate(t *testing.T) {
	t.Parallel()

	value := daemonSetStrategyValue(t, "OnDelete", "3", "4")
	strategy, diags := expandDaemonSetStrategyModel(context.Background(), value, path.Root("strategy"))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if strategy.Type != appsv1.OnDeleteDaemonSetStrategyType {
		t.Fatalf("expected strategy type %q, got %q", appsv1.OnDeleteDaemonSetStrategyType, strategy.Type)
	}
	if strategy.RollingUpdate != nil {
		t.Fatalf("expected nil rolling_update for OnDelete, got %#v", strategy.RollingUpdate)
	}
}

func TestExpandDaemonSetStrategy_RollingUpdateValues(t *testing.T) {
	t.Parallel()

	value := daemonSetStrategyValue(t, "RollingUpdate", "25%", "2")
	strategy, diags := expandDaemonSetStrategyModel(context.Background(), value, path.Root("strategy"))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if strategy.RollingUpdate == nil {
		t.Fatal("expected rolling_update to be present")
	}
	if got := strategy.RollingUpdate.MaxSurge.String(); got != "25%" {
		t.Fatalf("unexpected max_surge: got %q, want %q", got, "25%")
	}
	if got := strategy.RollingUpdate.MaxUnavailable.String(); got != "2" {
		t.Fatalf("unexpected max_unavailable: got %q, want %q", got, "2")
	}
}

func TestFlattenDaemonSetStrategy_DefaultTypeWhenMissing(t *testing.T) {
	t.Parallel()

	var diags diag.Diagnostics
	flattened := flattenDaemonSetStrategyModel(context.Background(), appsv1.DaemonSetUpdateStrategy{}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var models []DaemonSetStrategyModel
	diags.Append(flattened.ElementsAs(context.Background(), &models, false)...)
	if diags.HasError() || len(models) != 1 {
		t.Fatalf("decoding flattened strategy: %v (models=%d)", diags, len(models))
	}
	if got := models[0].Type.ValueString(); got != "RollingUpdate" {
		t.Fatalf("unexpected flattened type: got %q, want %q", got, "RollingUpdate")
	}
	if !models[0].RollingUpdate.IsNull() {
		t.Fatalf("expected rolling_update to be null, got %#v", models[0].RollingUpdate)
	}
}

func TestFlattenDaemonSetStrategy_DefaultRollingUpdateValues(t *testing.T) {
	t.Parallel()

	var diags diag.Diagnostics
	flattened := flattenDaemonSetStrategyModel(context.Background(), appsv1.DaemonSetUpdateStrategy{
		Type:          appsv1.RollingUpdateDaemonSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDaemonSet{},
	}, &diags)
	var models []DaemonSetStrategyModel
	diags.Append(flattened.ElementsAs(context.Background(), &models, false)...)
	var updates []DaemonSetRollingUpdateModel
	if len(models) == 1 {
		diags.Append(models[0].RollingUpdate.ElementsAs(context.Background(), &updates, false)...)
	}
	if diags.HasError() || len(updates) != 1 {
		t.Fatalf("decoding flattened strategy: %v (updates=%d)", diags, len(updates))
	}
	if got := updates[0].MaxSurge.ValueString(); got != "0" {
		t.Fatalf("unexpected flattened max_surge default: got %q, want %q", got, "0")
	}
	if got := updates[0].MaxUnavailable.ValueString(); got != "1" {
		t.Fatalf("unexpected flattened max_unavailable default: got %q, want %q", got, "1")
	}
}

func daemonSetStrategyValue(t *testing.T, strategyType, maxSurge, maxUnavailable string) types.List {
	t.Helper()
	rolling, diags := types.ListValueFrom(context.Background(), daemonSetRollingUpdateObjectType(), []DaemonSetRollingUpdateModel{{
		MaxSurge:       types.StringValue(maxSurge),
		MaxUnavailable: types.StringValue(maxUnavailable),
	}})
	if diags.HasError() {
		t.Fatalf("building rolling update: %v", diags)
	}
	value, diags := types.ListValueFrom(context.Background(), daemonSetStrategyObjectType(), []DaemonSetStrategyModel{{
		Type:          types.StringValue(strategyType),
		RollingUpdate: rolling,
	}})
	if diags.HasError() {
		t.Fatalf("building strategy: %v", diags)
	}
	return value
}
