// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestFlattenObjectRef(t *testing.T) {
	cases := []struct {
		Input          *v1.ObjectReference
		ExpectedOutput []interface{}
	}{
		{
			&v1.ObjectReference{
				Name:      "demo",
				Namespace: "default",
			},
			[]interface{}{
				map[string]interface{}{
					"name":      "demo",
					"namespace": "default",
				},
			},
		},
		{
			&v1.ObjectReference{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenObjectRef(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenNFSVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *v1.NFSVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&v1.NFSVolumeSource{
				Server:   "192.168.1.1",
				Path:     "/exports/data",
				ReadOnly: true,
			},
			[]interface{}{
				map[string]interface{}{
					"server":    "192.168.1.1",
					"path":      "/exports/data",
					"read_only": true,
				},
			},
		},
		{
			&v1.NFSVolumeSource{
				Server:   "192.168.1.2",
				Path:     "/exports/ro",
				ReadOnly: false,
			},
			[]interface{}{
				map[string]interface{}{
					"server":    "192.168.1.2",
					"path":      "/exports/ro",
					"read_only": false,
				},
			},
		},
		{
			&v1.NFSVolumeSource{},
			[]interface{}{
				map[string]interface{}{
					"server":    "",
					"path":      "",
					"read_only": false,
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenNFSVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
