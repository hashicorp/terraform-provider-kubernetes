// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestFlattenDataSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.TypedLocalObjectReference
		ExpectedOutput []interface{}
	}{
		{
			&corev1.TypedLocalObjectReference{
				APIGroup: ptr.To("snapshot.storage.k8s.io"),
				Kind:     "VolumeSnapshot",
				Name:     "my-snapshot",
			},
			[]interface{}{map[string]interface{}{
				"api_group": "snapshot.storage.k8s.io",
				"kind":      "VolumeSnapshot",
				"name":      "my-snapshot",
			}},
		},
		{
			&corev1.TypedLocalObjectReference{
				Kind: "PersistentVolumeClaim",
				Name: "source-pvc",
			},
			[]interface{}{map[string]interface{}{
				"kind": "PersistentVolumeClaim",
				"name": "source-pvc",
			}},
		},
	}

	for i, tc := range cases {
		output := flattenDataSource(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Case %d: unexpected result (-want +got):\n%s", i, diff)
		}
	}
}

func TestExpandDataSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.TypedLocalObjectReference
	}{
		{
			[]interface{}{map[string]interface{}{
				"api_group": "snapshot.storage.k8s.io",
				"kind":      "VolumeSnapshot",
				"name":      "my-snapshot",
			}},
			&corev1.TypedLocalObjectReference{
				APIGroup: ptr.To("snapshot.storage.k8s.io"),
				Kind:     "VolumeSnapshot",
				Name:     "my-snapshot",
			},
		},
		{
			[]interface{}{map[string]interface{}{
				"api_group": "",
				"kind":      "PersistentVolumeClaim",
				"name":      "source-pvc",
			}},
			&corev1.TypedLocalObjectReference{
				Kind: "PersistentVolumeClaim",
				Name: "source-pvc",
			},
		},
		{
			[]interface{}{},
			nil,
		},
		{
			[]interface{}{nil},
			nil,
		},
	}

	for i, tc := range cases {
		output := expandDataSource(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Case %d: unexpected result (-want +got):\n%s", i, diff)
		}
	}
}
