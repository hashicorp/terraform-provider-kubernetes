// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestFlattenTolerations(t *testing.T) {
	cases := []struct {
		Input          []corev1.Toleration
		isTemplate     bool
		ExpectedOutput []interface{}
	}{
		{
			[]corev1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
			false,
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
		},
		{
			[]corev1.Toleration{
				{
					Key:      "node-role.kubernetes.io/other-worker",
					Operator: "Exists",
				},
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
			false,
			[]interface{}{
				map[string]interface{}{
					"key":      "node-role.kubernetes.io/other-worker",
					"operator": "Exists",
				},
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
		},
		{
			[]corev1.Toleration{
				{
					Effect:            "NoExecute",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
			false,
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"toleration_seconds": "120",
				},
			},
		},
		{
			[]corev1.Toleration{
				{
					Effect:            "NoExecute",
					Key:               "node.kubernetes.io/unreachable",
					Operator:          "Exists",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
			false,
			[]interface{}{},
		},
		{
			[]corev1.Toleration{},
			false,
			[]interface{}{},
		},
		{
			[]corev1.Toleration{},
			true,
			[]interface{}{},
		},
		{
			[]corev1.Toleration{
				{
					Effect:            "NoExecute",
					Key:               "node.kubernetes.io/unreachable",
					Operator:          "Exists",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
			true,
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"key":                "node.kubernetes.io/unreachable",
					"operator":           "Exists",
					"toleration_seconds": "120",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenTolerations(tc.Input, tc.isTemplate)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandTolerations(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput []*corev1.Toleration
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
			[]*corev1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
				map[string]interface{}{
					"key":      "node-role.kubernetes.io/other-worker",
					"operator": "Exists",
				},
			},
			[]*corev1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
				{
					Key:      "node-role.kubernetes.io/other-worker",
					Operator: "Exists",
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"toleration_seconds": "120",
				},
			},
			[]*corev1.Toleration{
				{
					Effect:            "NoExecute",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
		},
		{
			[]interface{}{},
			[]*corev1.Toleration{},
		},
	}

	for _, tc := range cases {
		output, err := expandTolerations(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenSecretVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.SecretVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0644)),
				SecretName:  "secret1",
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"secret_name":  "secret1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
		},
		{
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0755)),
				SecretName:  "secret2",
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"secret_name":  "secret2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
		},
		{
			&corev1.SecretVolumeSource{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenSecretVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandSecretVolumeSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.SecretVolumeSource
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"secret_name":  "secret1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0644)),
				SecretName:  "secret1",
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"secret_name":  "secret2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0755)),
				SecretName:  "secret2",
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
		},
		{
			[]interface{}{},
			&corev1.SecretVolumeSource{},
		},
	}

	for _, tc := range cases {
		output, err := expandSecretVolumeSource(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenEmptyDirVolumeSource(t *testing.T) {
	size, _ := resource.ParseQuantity("64Mi")

	cases := []struct {
		Input          *corev1.EmptyDirVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
			[]interface{}{
				map[string]interface{}{
					"medium": "Memory",
				},
			},
		},
		{
			&corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumMemory,
				SizeLimit: &size,
			},
			[]interface{}{
				map[string]interface{}{
					"medium":     "Memory",
					"size_limit": "64Mi",
				},
			},
		},
		{
			&corev1.EmptyDirVolumeSource{},
			[]interface{}{
				map[string]interface{}{
					"medium": "",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenEmptyDirVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenConfigMapVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.ConfigMapVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap1",
				},
				DefaultMode: ptr.To(int32(0644)),
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"name":         "configmap1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
		},
		{
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap2",
				},
				DefaultMode: ptr.To(int32(0755)),
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"name":         "configmap2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
		},
		{
			&corev1.ConfigMapVolumeSource{},
			[]interface{}{map[string]interface{}{"name": ""}},
		},
	}

	for _, tc := range cases {
		output := flattenConfigMapVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandConfigMapVolumeSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.ConfigMapVolumeSource
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"name":         "configmap1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap1",
				},
				DefaultMode: ptr.To(int32(0644)),
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"name":         "configmap2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap2",
				},
				DefaultMode: ptr.To(int32(0755)),
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
		},
		{
			[]interface{}{},
			&corev1.ConfigMapVolumeSource{},
		},
	}

	for _, tc := range cases {
		output, err := expandConfigMapVolumeSource(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandThenFlatten_projected_volume(t *testing.T) {
	cases := []struct {
		Input *corev1.ProjectedVolumeSource
	}{
		{
			Input: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-1"},
						},
					},
					{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-1"},
						},
					},
					{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-2"},
						},
					},
					{
						DownwardAPI: &corev1.DownwardAPIProjection{
							Items: []corev1.DownwardAPIVolumeFile{
								{Path: "path-1"},
							},
						},
					},
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience: "audience-1",
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		in := tc.Input
		flattenedFirst := flattenProjectedVolumeSource(in)
		out, err := expandProjectedVolumeSource(flattenedFirst)
		if err != nil {
			t.Fatal(err)
		}
		if !cmp.Equal(in, out) {
			t.Fatal(cmp.Diff(in, out))
		}

		flattenedAgain := flattenProjectedVolumeSource(out)
		if !cmp.Equal(flattenedFirst, flattenedAgain) {
			t.Fatal(cmp.Diff(flattenedFirst, flattenedAgain))
		}
	}

}

func TestExpandCSIVolumeSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.CSIVolumeSource
	}{
		{
			Input: []interface{}{
				map[string]interface{}{
					"driver":    "secrets-store.csi.k8s.io",
					"read_only": true,
					"volume_attributes": map[string]interface{}{
						"secretProviderClass": "azure-keyvault",
					},
					"fs_type": "nfs",
					"node_publish_secret_ref": []interface{}{
						map[string]interface{}{
							"name": "secrets-store",
						},
					},
				},
			},
			ExpectedOutput: &corev1.CSIVolumeSource{
				Driver:   "secrets-store.csi.k8s.io",
				ReadOnly: ptr.To(true),
				FSType:   ptr.To("nfs"),
				VolumeAttributes: map[string]string{
					"secretProviderClass": "azure-keyvault",
				},
				NodePublishSecretRef: &corev1.LocalObjectReference{
					Name: "secrets-store",
				},
			},
		},
		{
			Input: []interface{}{
				map[string]interface{}{
					"driver": "other-csi-driver.k8s.io",
					"volume_attributes": map[string]interface{}{
						"objects": `array: 
						- |
							objectName: secret-1
							objectType: secret `,
					},
				},
			},
			ExpectedOutput: &corev1.CSIVolumeSource{
				Driver:   "other-csi-driver.k8s.io",
				ReadOnly: nil,
				FSType:   nil,
				VolumeAttributes: map[string]string{
					"objects": `array: 
						- |
							objectName: secret-1
							objectType: secret `,
				},
				NodePublishSecretRef: nil,
			},
		},
	}
	for _, tc := range cases {
		output := expandCSIVolumeSource(tc.Input)
		if !reflect.DeepEqual(tc.ExpectedOutput, output) {
			t.Fatalf("Unexpected output from CSI Volume Source Expander. \nExpected: %#v, Given: %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenCSIVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.CSIVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			Input: &corev1.CSIVolumeSource{
				Driver:   "secrets-store.csi.k8s.io",
				ReadOnly: ptr.To(true),
				FSType:   ptr.To("nfs"),
				VolumeAttributes: map[string]string{
					"secretProviderClass": "azure-keyvault",
				},
				NodePublishSecretRef: &corev1.LocalObjectReference{
					Name: "secrets-store",
				},
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"driver":    "secrets-store.csi.k8s.io",
					"read_only": true,
					"volume_attributes": map[string]string{
						"secretProviderClass": "azure-keyvault",
					},
					"fs_type": "nfs",
					"node_publish_secret_ref": []interface{}{
						map[string]interface{}{
							"name": "secrets-store",
						},
					},
				},
			},
		},
		{
			Input: &corev1.CSIVolumeSource{
				Driver:   "other-csi-driver.k8s.io",
				ReadOnly: nil,
				FSType:   nil,
				VolumeAttributes: map[string]string{
					"objects": `array: 
					- |
						objectName: secret-1
						objectType: secret `,
				},
				NodePublishSecretRef: nil,
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"driver": "other-csi-driver.k8s.io",
					"volume_attributes": map[string]string{
						"objects": `array: 
					- |
						objectName: secret-1
						objectType: secret `,
					},
				},
			},
		},
	}
	for _, tc := range cases {
		output := flattenCSIVolumeSource(tc.Input)
		if !reflect.DeepEqual(tc.ExpectedOutput, output) {
			t.Fatalf("Unexpected result from flattener. \nExpected %#v, \nGiven %#v",
				tc.ExpectedOutput, output)
		}
	}
}
