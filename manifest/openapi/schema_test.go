// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package openapi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestIsTypeFullyKnown(t *testing.T) {
	type testSample struct {
		s bool
		t tftypes.Type
	}

	type testSamples map[string]testSample

	samples := testSamples{
		"DynamicPseudoType": {
			s: false,
			t: tftypes.DynamicPseudoType,
		},
		"String": {
			s: true,
			t: tftypes.String,
		},
		"StringList": {
			s: true,
			t: tftypes.List{ElementType: tftypes.String},
		},
		"DynamicPseudoTypeList": {
			s: false,
			t: tftypes.List{ElementType: tftypes.DynamicPseudoType},
		},
		"DynamicPseudoTypeMap": {
			s: false,
			t: tftypes.Map{ElementType: tftypes.DynamicPseudoType},
		},
		"StringMap": {
			s: true,
			t: tftypes.Map{ElementType: tftypes.String},
		},
		"Object": {
			s: true,
			t: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.String,
					"bar": tftypes.Number,
				},
			},
		},
		"ObjectDynamic": {
			s: false,
			t: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.String,
					"bar": tftypes.DynamicPseudoType,
				},
			},
		},
		"ListObject": {
			s: true,
			t: tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.String,
					"bar": tftypes.Number,
				},
			},
			},
		},
		"ListObjectDynamic": {
			s: false,
			t: tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.String,
					"bar": tftypes.DynamicPseudoType,
				},
			},
			},
		},
	}

	for name, v := range samples {
		t.Run(name,
			func(t *testing.T) {
				if isTypeFullyKnown(v.t) != v.s {
					t.Fatalf("sample %s failed", name)
				}
			})
	}
}
