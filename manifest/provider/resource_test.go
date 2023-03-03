// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRemoveNulls(t *testing.T) {
	samples := []struct {
		in  map[string]interface{}
		out map[string]interface{}
	}{
		{
			in: map[string]interface{}{
				"foo": nil,
			},
			out: map[string]interface{}{},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": "test",
			},
			out: map[string]interface{}{
				"bar": "test",
			},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": []interface{}{nil, "test"},
			},
			out: map[string]interface{}{
				"bar": []interface{}{"test"},
			},
		},
		{
			in: map[string]interface{}{
				"foo": nil,
				"bar": []interface{}{
					map[string]interface{}{
						"some":  nil,
						"other": "data",
					},
					"test",
				},
			},
			out: map[string]interface{}{
				"bar": []interface{}{
					map[string]interface{}{
						"other": "data",
					},
					"test",
				},
			},
		},
	}

	for i, s := range samples {
		t.Run(fmt.Sprintf("sample%d", i+1), func(t *testing.T) {
			o := mapRemoveNulls(s.in)
			if !reflect.DeepEqual(s.out, o) {
				t.Fatal("sample and output are not equal")
			}
		})
	}
}
