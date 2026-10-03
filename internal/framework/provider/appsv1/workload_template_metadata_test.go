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
		all   bool
		want  types.Map
	}{
		{"refresh keeps every live key", live, planned, true, types.MapValueMust(types.StringType, map[string]attr.Value{
			"app": types.StringValue("x"), "kubectl.kubernetes.io/restartedAt": types.StringValue("now"),
		})},
		{"apply keeps planned keys", live, planned, false, planned},
		{"apply with null plan drops live keys", live, types.MapNull(types.StringType), false, types.MapNull(types.StringType)},
		{"apply with unknown plan keeps every live key", map[string]string{"app": "x"}, types.MapUnknown(types.StringType), false, planned},
		{"none stays null", nil, types.MapNull(types.StringType), true, types.MapNull(types.StringType)},
		{"none keeps an empty map", nil, empty, true, empty},
		{"removed keys leave an empty map", nil, planned, true, empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := workloadTemplateMetadataMap(context.Background(), tc.live, tc.prior, tc.all)
			if diags.HasError() || !got.Equal(tc.want) {
				t.Fatalf("got %s (%v), want %s", got, diags, tc.want)
			}
		})
	}
}
