// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

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
