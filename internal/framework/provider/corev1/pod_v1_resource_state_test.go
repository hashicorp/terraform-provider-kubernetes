// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestPodResourcesStateRoutes(t *testing.T) {
	ctx := context.Background()
	pod := &PodV1{}
	var schemaResponse resource.SchemaResponse
	pod.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	for _, version := range []int64{0, 1} {
		for _, move := range []bool{false, true} {
			t.Run(fmt.Sprintf("v%d/move=%t", version, move), func(t *testing.T) {
				rawQuantities := `{"limits":{"cpu":"0.5"},"requests":null}`
				if version == 0 {
					rawQuantities = `{"limits":[{"cpu":"0.5"}],"requests":[]}`
				}
				raw := &tfprotov6.RawState{JSON: []byte(`{"id":"ns/p","metadata":[{"name":"p","namespace":"ns","generate_name":""}],"target_state":null,"timeouts":{"create":""},"spec":[{"container":[{"name":"app","resources":[` + rawQuantities + `]}],"init_container":[{"name":"init","resources":[]}]}]}`)}
				state := tfsdk.State{Schema: schemaResponse.Schema}
				if move {
					response := resource.MoveStateResponse{TargetState: state}
					pod.MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{SourceTypeName: "kubernetes_pod", SourceSchemaVersion: version, SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceRawState: raw}, &response)
					if response.Diagnostics.HasError() {
						t.Fatal(response.Diagnostics)
					}
					state = response.TargetState
				} else {
					response := resource.UpgradeStateResponse{State: state}
					pod.UpgradeState(ctx)[version].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: raw}, &response)
					if response.Diagnostics.HasError() {
						t.Fatal(response.Diagnostics)
					}
					state = response.State
				}
				at := path.Root("spec").AtListIndex(0)
				var quantities types.Object
				if d := state.GetAttribute(ctx, at.AtName("container").AtListIndex(0).AtName("resources"), &quantities); d.HasError() {
					t.Fatal(d)
				}
				limits := quantities.Attributes()["limits"].(types.Map)
				if !limits.Elements()["cpu"].Equal(types.StringValue("0.5")) {
					t.Fatalf("quantity spelling changed: %s", limits)
				}
				requests := quantities.Attributes()["requests"].(types.Map)
				if requests.IsNull() != (version == 1) {
					t.Fatalf("requests = %s", requests)
				}
				var zero types.Object
				if d := state.GetAttribute(ctx, at.AtName("init_container").AtListIndex(0).AtName("resources"), &zero); d.HasError() {
					t.Fatal(d)
				}
				for _, key := range []string{"limits", "requests"} {
					if !zero.Attributes()[key].Equal(types.MapValueMust(types.StringType, map[string]attr.Value{})) {
						t.Fatalf("zero resources = %s", zero)
					}
				}
				var id types.String
				var target types.List
				if d := state.GetAttribute(ctx, path.Root("id"), &id); d.HasError() {
					t.Fatal(d)
				}
				if d := state.GetAttribute(ctx, path.Root("target_state"), &target); d.HasError() {
					t.Fatal(d)
				}
				if id.ValueString() != "ns/p" || target.IsNull() || len(target.Elements()) != 0 {
					t.Fatalf("identity/default repair lost: %s %s", id, target)
				}
			})
		}
	}
}

func TestPodLegacyStateSpecValidation(t *testing.T) {
	ctx := context.Background()
	pod := &PodV1{}
	var schemaResponse resource.SchemaResponse
	pod.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	for _, version := range []int64{0, 1} {
		for _, move := range []bool{false, true} {
			for _, tc := range []struct {
				name, field string
				invalid     bool
				missing     bool
			}{
				{"absent", "", false, true},
				{"null", `,"spec":null`, false, true},
				{"empty", `,"spec":[]`, false, true},
				{"one object", `,"spec":[{}]`, false, false},
				{"multiple objects", `,"spec":[{},{}]`, true, false},
				{"null element", `,"spec":[null]`, true, false},
			} {
				t.Run(fmt.Sprintf("v%d/move=%t/%s", version, move, tc.name), func(t *testing.T) {
					raw := &tfprotov6.RawState{JSON: []byte(`{"id":"ns/p","metadata":[{"name":"p","namespace":"ns"}]` + tc.field + `}`)}
					state := tfsdk.State{Schema: schemaResponse.Schema}
					wantError := tc.invalid || version == 0 && tc.missing
					if move {
						response := resource.MoveStateResponse{TargetState: state}
						pod.MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
							SourceTypeName: "kubernetes_pod", SourceSchemaVersion: version,
							SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceRawState: raw,
						}, &response)
						if response.Diagnostics.HasError() != wantError {
							t.Fatalf("diagnostics = %v, want error = %t", response.Diagnostics, wantError)
						}
					} else {
						response := resource.UpgradeStateResponse{State: state}
						pod.UpgradeState(ctx)[version].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: raw}, &response)
						if response.Diagnostics.HasError() != wantError {
							t.Fatalf("diagnostics = %v, want error = %t", response.Diagnostics, wantError)
						}
					}
				})
			}
		}
	}
}
