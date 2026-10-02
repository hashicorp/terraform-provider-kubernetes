// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func TestJobPodFailurePolicyReplacementRules(t *testing.T) {
	ctx := context.Background()
	legacySpec := kubernetes.Provider().ResourcesMap["kubernetes_job"].Schema["spec"].Elem.(*sdkschema.Resource).Schema
	fields := map[string]*sdkschema.Schema{"policy": legacySpec["pod_failure_policy"]}
	policy := func(action string) []interface{} {
		if action == "" {
			return []interface{}{}
		}
		return []interface{}{map[string]interface{}{
			"rule": []interface{}{map[string]interface{}{
				"action": action,
				"on_exit_codes": []interface{}{map[string]interface{}{
					"operator": "In", "values": []interface{}{1},
				}},
			}},
		}}
	}
	for _, test := range []struct {
		name, before, after string
		replace             bool
	}{
		{"same-count rule change", "Ignore", "Count", false},
		{"policy added", "", "Count", true},
		{"policy removed", "Ignore", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, after := policy(test.before), policy(test.after)
			data := sdkschema.TestResourceDataRaw(t, fields, map[string]interface{}{"policy": before})
			data.SetId("existing")
			diff, err := sdkschema.InternalMap(fields).Diff(ctx, data.State(),
				terraform.NewResourceConfigRaw(map[string]interface{}{"policy": after}), nil, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			legacyReplace := diff != nil && diff.RequiresNew()
			if legacyReplace != test.replace {
				t.Fatalf("SDK replacement=%t, expected %t", legacyReplace, test.replace)
			}
			for _, updatable := range []bool{false, true} {
				block := jobSpecBlock(updatable).NestedObject.Blocks["pod_failure_policy"].(schema.ListNestedBlock)
				value := func(raw []interface{}) types.List {
					encoded, err := json.Marshal(raw)
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := (&tfprotov6.RawState{JSON: encoded}).Unmarshal(block.Type().TerraformType(ctx))
					if err != nil {
						t.Fatal(err)
					}
					result, err := block.Type().ValueFromTerraform(ctx, decoded)
					if err != nil {
						t.Fatal(err)
					}
					return result.(types.List)
				}
				request := planmodifier.ListRequest{
					State:      tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
					Plan:       tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
					StateValue: value(before), PlanValue: value(after), ConfigValue: value(after),
				}
				response := planmodifier.ListResponse{PlanValue: request.PlanValue}
				for _, modifier := range block.PlanModifiers {
					modifier.PlanModifyList(ctx, request, &response)
				}
				// Job policy edits must replace an immutable API object, rather
				// than retain the SDK's ineffective same-count in-place update.
				expectedReplace := legacyReplace || (!updatable && test.before != test.after)
				if response.Diagnostics.HasError() || response.RequiresReplace != expectedReplace {
					t.Fatalf("updatable=%t: Framework replacement=%t, expected=%t (SDK=%t); %s",
						updatable, response.RequiresReplace, expectedReplace, legacyReplace, response.Diagnostics)
				}
			}
		})
	}
}
