// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	networking "k8s.io/api/networking/v1"
)

// A mutating admission policy must not change known planned values.
func TestNetworkPolicyWritePreservesPlan(t *testing.T) {
	ctx := context.Background()
	var object *networking.NetworkPolicy
	r := networkPolicyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPost {
			if err := json.NewDecoder(req.Body).Decode(&object); err != nil {
				t.Fatal(err)
			}
			object.APIVersion, object.Kind = networkPolicyAPIVersion, networkPolicyKind
			object.UID, object.ResourceVersion, object.Generation = "uid-1", "10", 1
			object.Spec.PodSelector.MatchLabels = map[string]string{"admission.example/tenant": "one"}
		}
		if err := json.NewEncoder(w).Encode(object); err != nil {
			t.Error(err)
		}
	})
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	raw, _, _, err := decodeNetworkingState(ctx, &tfprotov6.RawState{JSON: []byte(networkPolicyTestStoredState)}, schema.Schema.Type().TerraformType(ctx), "network_policy")
	if err != nil {
		t.Fatal(err)
	}
	initial := tfsdk.State{Schema: schema.Schema, Raw: raw}
	response := resource.CreateResponse{State: tfsdk.State{Schema: initial.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: initial.Schema, Raw: initial.Raw}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var plan, actual NetworkPolicyV1Model
	if d := initial.Get(ctx, &plan); d.HasError() {
		t.Fatal(d)
	}
	if d := response.State.Get(ctx, &actual); d.HasError() {
		t.Fatal(d)
	}
	if !plan.Spec[0].PodSelector.MatchLabels.Equal(actual.Spec[0].PodSelector.MatchLabels) {
		t.Fatalf("known planned match_labels changed during Create: plan=%s actual=%s", plan.Spec[0].PodSelector.MatchLabels, actual.Spec[0].PodSelector.MatchLabels)
	}
	// An update response can also contain admission changes. Keep the new plan,
	// then expose the API value as drift on Read.
	prior := response.State
	plan.Spec[0].PodSelector.MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{"configured": types.StringValue("new")})
	proposed := tfsdk.State{Schema: initial.Schema}
	if d := proposed.Set(ctx, &plan); d.HasError() {
		t.Fatal(d)
	}
	update := resource.UpdateResponse{State: prior}
	r.Update(ctx, resource.UpdateRequest{State: prior, Plan: tfsdk.Plan{Schema: proposed.Schema, Raw: proposed.Raw}}, &update)
	if update.Diagnostics.HasError() {
		t.Fatal(update.Diagnostics)
	}
	if d := update.State.Get(ctx, &actual); d.HasError() {
		t.Fatal(d)
	}
	if !actual.Spec[0].PodSelector.MatchLabels.Equal(plan.Spec[0].PodSelector.MatchLabels) {
		t.Fatal("Update changed the planned selector")
	}
	read := resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if d := read.State.Get(ctx, &actual); d.HasError() {
		t.Fatal(d)
	}
	if actual.Spec[0].PodSelector.MatchLabels.Equal(plan.Spec[0].PodSelector.MatchLabels) {
		t.Fatal("Read hid admission drift")
	}

}
