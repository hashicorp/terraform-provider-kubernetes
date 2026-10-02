// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDaemonSetTemplateMetadataExplicitEmptyMaps(t *testing.T) {
	prior := []common.NamespacedMetadataModel{{
		MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
			Annotations: types.MapValueMust(types.StringType, nil),
			Labels:      types.MapValueMust(types.StringType, nil),
		}},
	}}
	got, diags := flattenDaemonSetTemplateMetadata(context.Background(), metav1.ObjectMeta{}, prior)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got[0].Annotations.Equal(prior[0].Annotations) || !got[0].Labels.Equal(prior[0].Labels) {
		t.Fatalf("explicit empty maps became annotations=%s labels=%s", got[0].Annotations, got[0].Labels)
	}
}

func TestDaemonSetV0StateValidationRejectsMalformedNestedValues(t *testing.T) {
	t.Parallel()

	testCases := []map[string]interface{}{
		{"spec": []interface{}{"not-an-object"}},
		{"spec": []interface{}{map[string]interface{}{"template": []interface{}{"not-an-object"}}}},
		{"spec": []interface{}{map[string]interface{}{"template": []interface{}{map[string]interface{}{
			"spec": []interface{}{"not-an-object"},
		}}}}},
		{"spec": []interface{}{map[string]interface{}{"template": []interface{}{map[string]interface{}{
			"spec": []interface{}{map[string]interface{}{"container": []interface{}{"not-an-object"}}},
		}}}}},
	}
	for i, raw := range testCases {
		if err := validateDaemonSetV0State(raw); err == nil {
			t.Errorf("case %d: expected malformed state error", i)
		}
	}
}

func TestDaemonSetMoveStateMalformedV0DoesNotPanic(t *testing.T) {
	t.Parallel()

	movers := (&DaemonSetV1{}).MoveState(context.Background())
	if len(movers) != 1 {
		t.Fatalf("MoveState returned %d movers, want 1", len(movers))
	}
	response := &resource.MoveStateResponse{}
	movers[0].StateMover(context.Background(), resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        "kubernetes_daemonset",
		SourceSchemaVersion:   0,
		SourceRawState: &tfprotov6.RawState{
			JSON: []byte(`{"spec":["malformed"]}`),
		},
	}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for malformed v0 state")
	}
}

func TestDaemonSetImportRejectsMalformedID(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"name-only", "/name", "namespace/", "too/many/parts"} {
		response := &resource.ImportStateResponse{}
		(&DaemonSetV1{}).ImportState(context.Background(), resource.ImportStateRequest{ID: id}, response)
		if !response.Diagnostics.HasError() {
			t.Errorf("ImportState(%q) did not return an error", id)
		} else if !strings.Contains(response.Diagnostics[0].Summary(), "Invalid import ID") {
			t.Errorf("ImportState(%q) summary = %q", id, response.Diagnostics[0].Summary())
		}
	}
}

func TestDaemonSetNoOpPlanAddedComputedBlock(t *testing.T) {
	objectType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name": tftypes.String, "computed": tftypes.String,
	}}
	listType := tftypes.List{ElementType: objectType}
	entry := func(name string, computed interface{}) tftypes.Value {
		return tftypes.NewValue(objectType, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, name), "computed": tftypes.NewValue(tftypes.String, computed),
		})
	}
	config := tftypes.NewValue(listType, []tftypes.Value{entry("new", nil)})
	plan := tftypes.NewValue(listType, []tftypes.Value{entry("new", tftypes.UnknownValue)})
	state := tftypes.NewValue(listType, []tftypes.Value{})
	got, usePrior, err := workloadNoOpPlan(config, plan, state)
	if err != nil {
		t.Fatalf("adding a block with computed children failed: %s", err)
	}
	if usePrior || !got.Equal(plan) {
		t.Fatalf("new block was hidden: usePrior=%t plan=%s", usePrior, got)
	}
}

func TestDaemonSetNoOpPlan(t *testing.T) {
	t.Parallel()

	valueType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"configured": tftypes.String,
		"computed":   tftypes.String,
	}}
	value := func(configured, computed interface{}) tftypes.Value {
		return tftypes.NewValue(valueType, map[string]tftypes.Value{
			"configured": tftypes.NewValue(tftypes.String, configured),
			"computed":   tftypes.NewValue(tftypes.String, computed),
		})
	}

	state := value("same", "prior")
	config := value("same", nil)
	plan := value("same", tftypes.UnknownValue)

	got, usePrior, err := workloadNoOpPlan(config, plan, state)
	if err != nil {
		t.Fatal(err)
	}
	if !usePrior || !got.Equal(state) {
		t.Fatalf("plan was not normalized to prior state: usePrior=%t plan=%s", usePrior, got)
	}

	changed := value("changed", tftypes.UnknownValue)
	got, usePrior, err = workloadNoOpPlan(config, changed, state)
	if err != nil {
		t.Fatal(err)
	}
	if usePrior || !got.Equal(changed) {
		t.Fatalf("known configuration change was hidden: usePrior=%t plan=%s", usePrior, got)
	}

	unknownConfig := value(tftypes.UnknownValue, nil)
	got, usePrior, err = workloadNoOpPlan(unknownConfig, plan, state)
	if err != nil {
		t.Fatal(err)
	}
	if usePrior || !got.Equal(plan) {
		t.Fatalf("unknown configuration was normalized: usePrior=%t plan=%s", usePrior, got)
	}
}
