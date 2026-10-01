// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

func TestEmptyMetadataCompatibilityIsWorkloadOptIn(t *testing.T) {
	ctx := context.Background()
	var namespace resource.SchemaResponse
	corev1.NewNamespaceV1().Schema(ctx, resource.SchemaRequest{}, &namespace)
	if namespace.Diagnostics.HasError() {
		t.Fatal(namespace.Diagnostics)
	}
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
	empty := types.MapValueMust(types.StringType, nil)
	null := types.MapNull(types.StringType)

	for name, tc := range map[string]struct {
		metadata schema.ListNestedBlock
		retain   bool
	}{
		"namespace": {
			metadata: namespace.Schema.Blocks["metadata"].(schema.ListNestedBlock),
		},
		"workload": {
			metadata: common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("workload", true)),
			retain:   true,
		},
	} {
		for _, field := range []string{"annotations", "labels"} {
			t.Run(name+"/"+field, func(t *testing.T) {
				attribute := tc.metadata.NestedObject.Attributes[field].(schema.MapAttribute)
				if !attribute.Optional || attribute.Computed != tc.retain || (attribute.Default != nil) != tc.retain {
					t.Fatalf("unexpected collection ownership: optional=%t computed=%t default=%T",
						attribute.Optional, attribute.Computed, attribute.Default)
				}
				response := planmodifier.MapResponse{PlanValue: null}
				for _, modifier := range attribute.PlanModifiers {
					modifier.PlanModifyMap(ctx, planmodifier.MapRequest{
						ConfigValue: null, StateValue: empty, PlanValue: response.PlanValue,
						Plan: tfsdk.Plan{Raw: raw}, State: tfsdk.State{Raw: raw},
					}, &response)
				}
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				want := null
				if tc.retain {
					want = empty
				}
				if !response.PlanValue.Equal(want) {
					t.Fatalf("empty then omitted: got %s, want %s", response.PlanValue, want)
				}
			})
		}
	}
}
