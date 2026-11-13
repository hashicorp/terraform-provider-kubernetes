// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFlattenResourceList_emptyPreservation(t *testing.T) {
	ctx := context.Background()
	emptyMap, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	if diags.HasError() {
		t.Fatalf("failed to create empty map: %s", diags)
	}
	nullMap := types.MapNull(types.StringType)

	cases := []struct {
		name      string
		list      corev1api.ResourceList
		prior     types.Map
		wantNull  bool
		wantEmpty bool
	}{
		{
			name:      "empty list with null prior returns null",
			list:      corev1api.ResourceList{},
			prior:     nullMap,
			wantNull:  true,
			wantEmpty: false,
		},
		{
			name:      "nil list with null prior returns null",
			list:      nil,
			prior:     nullMap,
			wantNull:  true,
			wantEmpty: false,
		},
		{
			name:      "empty list with empty map prior returns empty map",
			list:      corev1api.ResourceList{},
			prior:     emptyMap,
			wantNull:  false,
			wantEmpty: true,
		},
		{
			name: "populated list returns map regardless of prior",
			list: corev1api.ResourceList{
				corev1api.ResourceCPU: resource.MustParse("200m"),
			},
			prior:     nullMap,
			wantNull:  false,
			wantEmpty: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, d := flattenResourceList(ctx, tc.list, tc.prior)
			if d.HasError() {
				t.Fatalf("unexpected diagnostics: %s", d)
			}
			if tc.wantNull && !got.IsNull() {
				t.Errorf("got %s, want null", got)
			}
			if tc.wantEmpty {
				if got.IsNull() {
					t.Errorf("got null, want empty map")
				} else if len(got.Elements()) != 0 {
					t.Errorf("got %d elements, want 0", len(got.Elements()))
				}
			}
			if !tc.wantNull && !tc.wantEmpty {
				if got.IsNull() || len(got.Elements()) == 0 {
					t.Errorf("got %s, want populated map", got)
				}
			}
		})
	}
}

func TestFlattenLimitRangeSpec_emptyPreservation(t *testing.T) {
	ctx := context.Background()
	emptyMap, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	nullMap := types.MapNull(types.StringType)

	spec := corev1api.LimitRangeSpec{
		Limits: []corev1api.LimitRangeItem{
			{
				Type: corev1api.LimitTypeContainer,
				Default: corev1api.ResourceList{
					corev1api.ResourceCPU: resource.MustParse("200m"),
				},
				Min: corev1api.ResourceList{},
			},
		},
	}

	priorWithEmpty := []LimitRangeSpecModel{
		{
			Limit: []LimitRangeItemModel{
				{
					Type:    types.StringValue("Container"),
					Default: types.MapNull(types.StringType),
					Min:     emptyMap,
				},
			},
		},
	}

	priorWithNull := []LimitRangeSpecModel{
		{
			Limit: []LimitRangeItemModel{
				{
					Type:    types.StringValue("Container"),
					Default: types.MapNull(types.StringType),
					Min:     nullMap,
				},
			},
		},
	}

	// Case 1: prior had empty map for Min -> flattened model keeps empty map for Min
	got, d := flattenLimitRangeSpec(ctx, spec, priorWithEmpty)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %s", d)
	}
	if len(got) == 0 || len(got[0].Limit) == 0 {
		t.Fatalf("unexpected empty flattened spec: %v", got)
	}
	if got[0].Limit[0].Min.IsNull() {
		t.Errorf("Min is null, want empty map")
	} else if len(got[0].Limit[0].Min.Elements()) != 0 {
		t.Errorf("Min has %d elements, want 0", len(got[0].Limit[0].Min.Elements()))
	}

	// Case 2: prior had null for Min -> flattened model keeps null for Min
	got, d = flattenLimitRangeSpec(ctx, spec, priorWithNull)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %s", d)
	}
	if len(got) == 0 || len(got[0].Limit) == 0 {
		t.Fatalf("unexpected empty flattened spec: %v", got)
	}
	if !got[0].Limit[0].Min.IsNull() {
		t.Errorf("Min is %s, want null", got[0].Limit[0].Min)
	}
}
