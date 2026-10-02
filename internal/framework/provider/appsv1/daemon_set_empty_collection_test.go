// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
)

func TestDaemonSetEmptyCollectionPlanModifiers(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	(&DaemonSetV1{}).Schema(ctx, resource.SchemaRequest{}, &response)
	metadata := response.Schema.Blocks["metadata"].(schema.ListNestedBlock).NestedObject.Attributes
	templateMetadata := daemonSetTemplateMetadataSchema().NestedObject.Attributes
	pod := podspec.SpecBlock(podspec.Options{}).NestedObject
	container := pod.Blocks["container"].(schema.ListNestedBlock).NestedObject
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})

	for name, attribute := range map[string]schema.Attribute{
		"annotations":          metadata["annotations"],
		"labels":               metadata["labels"],
		"template annotations": templateMetadata["annotations"],
		"template labels":      templateMetadata["labels"],
		"node_selector":        pod.Attributes["node_selector"],
		"args":                 container.Attributes["args"],
		"command":              container.Attributes["command"],
	} {
		t.Run(name, func(t *testing.T) {
			var empty, nonempty, null, unknown attr.Value
			switch attribute.(type) {
			case schema.MapAttribute:
				empty = types.MapValueMust(types.StringType, nil)
				nonempty = types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")})
				null, unknown = types.MapNull(types.StringType), types.MapUnknown(types.StringType)
			case schema.ListAttribute:
				empty = types.ListValueMust(types.StringType, nil)
				nonempty = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("value")})
				null, unknown = types.ListNull(types.StringType), types.ListUnknown(types.StringType)
			default:
				t.Fatalf("unexpected attribute %T", attribute)
			}
			for caseName, tc := range map[string]struct {
				config, prior, proposed, want attr.Value
				create                        bool
				destroy                       bool
			}{
				"legacy empty":      {null, empty, null, empty, false, false},
				"remove nonempty":   {null, nonempty, nonempty, null, false, false},
				"unknown input":     {unknown, empty, unknown, unknown, false, false},
				"fresh omission":    {null, null, unknown, null, true, false},
				"explicit empty":    {empty, nonempty, empty, empty, false, false},
				"explicit nonempty": {nonempty, empty, nonempty, nonempty, false, false},
				"destroy":           {null, empty, null, null, false, true},
			} {
				t.Run(caseName, func(t *testing.T) {
					plan := tfsdk.Plan{Raw: raw}
					state := tfsdk.State{Raw: raw}
					if tc.create {
						state.Raw = tftypes.NewValue(raw.Type(), nil)
					}
					if tc.destroy {
						plan.Raw = tftypes.NewValue(raw.Type(), nil)
					}
					got := tc.proposed
					switch a := attribute.(type) {
					case schema.MapAttribute:
						resp := planmodifier.MapResponse{PlanValue: got.(types.Map)}
						for _, modifier := range a.PlanModifiers {
							modifier.PlanModifyMap(ctx, planmodifier.MapRequest{
								ConfigValue: tc.config.(types.Map), StateValue: tc.prior.(types.Map),
								PlanValue: resp.PlanValue, Plan: plan, State: state,
							}, &resp)
						}
						if resp.Diagnostics.HasError() {
							t.Fatal(resp.Diagnostics)
						}
						got = resp.PlanValue
					case schema.ListAttribute:
						resp := planmodifier.ListResponse{PlanValue: got.(types.List)}
						for _, modifier := range a.PlanModifiers {
							modifier.PlanModifyList(ctx, planmodifier.ListRequest{
								ConfigValue: tc.config.(types.List), StateValue: tc.prior.(types.List),
								PlanValue: resp.PlanValue, Plan: plan, State: state,
							}, &resp)
						}
						if resp.Diagnostics.HasError() {
							t.Fatal(resp.Diagnostics)
						}
						got = resp.PlanValue
					}
					if !got.Equal(tc.want) {
						t.Fatalf("planned %s, want %s", got, tc.want)
					}
				})
			}
		})
	}
}
