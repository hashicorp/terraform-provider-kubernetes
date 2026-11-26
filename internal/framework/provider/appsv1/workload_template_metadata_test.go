// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkloadTemplateMetadataMap(t *testing.T) {
	live := map[string]string{"app": "x", "kubectl.kubernetes.io/restartedAt": "now"}
	planned := types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("x")})
	empty := types.MapValueMust(types.StringType, nil)
	for _, tc := range []struct {
		name  string
		live  map[string]string
		prior types.Map
		want  types.Map
	}{
		{"keeps every live key", live, planned, types.MapValueMust(types.StringType, map[string]attr.Value{
			"app": types.StringValue("x"), "kubectl.kubernetes.io/restartedAt": types.StringValue("now"),
		})},
		{"none stays null", nil, types.MapNull(types.StringType), types.MapNull(types.StringType)},
		{"none keeps an empty map", nil, empty, empty},
		{"removed keys leave an empty map", nil, planned, empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := workloadTemplateMetadataMap(context.Background(), tc.live, tc.prior)
			if diags.HasError() || !got.Equal(tc.want) {
				t.Fatalf("got %s (%v), want %s", got, diags, tc.want)
			}
		})
	}
}
