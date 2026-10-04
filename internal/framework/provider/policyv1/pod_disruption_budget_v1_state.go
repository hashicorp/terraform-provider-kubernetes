// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// UpgradeState converts the released SDKv2 selector block directly to the current
// object schema. Same-type upgrades do not require a moved block.
func (r *PodDisruptionBudgetV1) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			if req.RawState == nil || len(req.RawState.JSON) == 0 {
				resp.Diagnostics.AddError("Unable to upgrade PodDisruptionBudget state", "The source state has no JSON data.")
				return
			}
			data, err := upgradePDBSelectorState(req.RawState.JSON)
			if err != nil {
				resp.Diagnostics.AddError("Unable to upgrade PodDisruptionBudget state", err.Error()+" Restore a valid state backup, or verify the existing Kubernetes object and re-import it using its namespace/name ID. An absent selector cannot safely be converted to an empty object because that selects all pods in the namespace.")
				return
			}
			value, err := (&tfprotov6.RawState{JSON: data}).Unmarshal(resp.State.Schema.Type().TerraformType(ctx))
			if err != nil {
				resp.Diagnostics.AddError("Unable to decode upgraded PodDisruptionBudget state", err.Error())
				return
			}
			resp.State.Raw = value
		}},
	}
}

// Convert the selector shape and the unset generate_name representation that
// Read writes. RawMessage preserves other values, including explicit empty
// collections. Accepting an object makes the conversion idempotent.
func upgradePDBSelectorState(data []byte) ([]byte, error) {
	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("the source state must be an object")
	}
	var specs []map[string]json.RawMessage
	if err := json.Unmarshal(state["spec"], &specs); err != nil || len(specs) != 1 || specs[0] == nil {
		return nil, fmt.Errorf("spec must contain exactly one object")
	}
	selector := specs[0]["selector"]
	var object map[string]json.RawMessage
	if err := json.Unmarshal(selector, &object); err != nil || object == nil {
		var legacy []map[string]json.RawMessage
		if err := json.Unmarshal(selector, &legacy); err != nil || len(legacy) != 1 || legacy[0] == nil {
			return nil, fmt.Errorf("spec[0].selector must be an object or a legacy list containing exactly one object")
		}
		object = legacy[0]
	}
	var err error
	specs[0]["selector"], err = json.Marshal(object)
	if err != nil {
		return nil, err
	}
	state["spec"], err = json.Marshal(specs)
	if err != nil {
		return nil, err
	}
	var metadata []map[string]json.RawMessage
	if err := json.Unmarshal(state["metadata"], &metadata); err != nil {
		return nil, err
	}
	if len(metadata) == 1 && bytes.Equal(bytes.TrimSpace(metadata[0]["generate_name"]), []byte(`""`)) {
		metadata[0]["generate_name"] = json.RawMessage(`null`)
		state["metadata"], err = json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(state)
}
