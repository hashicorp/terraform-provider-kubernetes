package v1

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFlattenTolerations(t *testing.T) {
	cases := []struct {
		Input          []v1.Toleration
		ExpectedOutput []interface{}
	}{
		{
			[]v1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
		},
		{
			[]v1.Toleration{
				{
					Key:      "node-role.kubernetes.io/other-worker",
					Operator: "Exists",
				},
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
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
			[]v1.Toleration{
				{
					Effect:            "NoExecute",
					TolerationSeconds: structures.PtrToInt64(120),
				},
			},
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"toleration_seconds": "120",
				},
			},
		},
		{
			[]v1.Toleration{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenTolerations(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandTolerations(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput []*v1.Toleration
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
			[]*v1.Toleration{
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
			[]*v1.Toleration{
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
			[]*v1.Toleration{
				{
					Effect:            "NoExecute",
					TolerationSeconds: structures.PtrToInt64(120),
				},
			},
		},
		{
			[]interface{}{},
			[]*v1.Toleration{},
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
		Input          *v1.SecretVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&v1.SecretVolumeSource{
				DefaultMode: structures.PtrToInt32(0644),
				SecretName:  "secret1",
				Optional:    structures.PtrToBool(true),
				Items: []v1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: structures.PtrToInt32(0600),
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
			&v1.SecretVolumeSource{
				DefaultMode: structures.PtrToInt32(0755),
				SecretName:  "secret2",
				Items: []v1.KeyToPath{
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
			&v1.SecretVolumeSource{},
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
		ExpectedOutput *v1.SecretVolumeSource
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
			&v1.SecretVolumeSource{
				DefaultMode: structures.PtrToInt32(0644),
				SecretName:  "secret1",
				Optional:    structures.PtrToBool(true),
				Items: []v1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: structures.PtrToInt32(0600),
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
			&v1.SecretVolumeSource{
				DefaultMode: structures.PtrToInt32(0755),
				SecretName:  "secret2",
				Items: []v1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
		},
		{
			[]interface{}{},
			&v1.SecretVolumeSource{},
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
		Input          *v1.EmptyDirVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&v1.EmptyDirVolumeSource{
				Medium: v1.StorageMediumMemory,
			},
			[]interface{}{
				map[string]interface{}{
					"medium": "Memory",
				},
			},
		},
		{
			&v1.EmptyDirVolumeSource{
				Medium:    v1.StorageMediumMemory,
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
			&v1.EmptyDirVolumeSource{},
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
		Input          *v1.ConfigMapVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap1",
				},
				DefaultMode: structures.PtrToInt32(0644),
				Optional:    structures.PtrToBool(true),
				Items: []v1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: structures.PtrToInt32(0600),
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
			&v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap2",
				},
				DefaultMode: structures.PtrToInt32(0755),
				Items: []v1.KeyToPath{
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
			&v1.ConfigMapVolumeSource{},
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
		ExpectedOutput *v1.ConfigMapVolumeSource
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
			&v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap1",
				},
				DefaultMode: structures.PtrToInt32(0644),
				Optional:    structures.PtrToBool(true),
				Items: []v1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: structures.PtrToInt32(0600),
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
			&v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap2",
				},
				DefaultMode: structures.PtrToInt32(0755),
				Items: []v1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
		},
		{
			[]interface{}{},
			&v1.ConfigMapVolumeSource{},
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
		Input *v1.ProjectedVolumeSource
	}{
		{
			Input: &v1.ProjectedVolumeSource{
				Sources: []v1.VolumeProjection{
					{
						Secret: &v1.SecretProjection{
							LocalObjectReference: v1.LocalObjectReference{Name: "secret-1"},
						},
					},
					{
						ConfigMap: &v1.ConfigMapProjection{
							LocalObjectReference: v1.LocalObjectReference{Name: "config-1"},
						},
					},
					{
						ConfigMap: &v1.ConfigMapProjection{
							LocalObjectReference: v1.LocalObjectReference{Name: "config-2"},
						},
					},
					{
						DownwardAPI: &v1.DownwardAPIProjection{
							Items: []v1.DownwardAPIVolumeFile{
								{Path: "path-1"},
							},
						},
					},
					{
						ServiceAccountToken: &v1.ServiceAccountTokenProjection{
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
		ExpectedOutput *v1.CSIVolumeSource
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
			ExpectedOutput: &v1.CSIVolumeSource{
				Driver:   "secrets-store.csi.k8s.io",
				ReadOnly: structures.PtrToBool(true),
				FSType:   structures.PtrToString("nfs"),
				VolumeAttributes: map[string]string{
					"secretProviderClass": "azure-keyvault",
				},
				NodePublishSecretRef: &v1.LocalObjectReference{
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
			ExpectedOutput: &v1.CSIVolumeSource{
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
		Input          *v1.CSIVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			Input: &v1.CSIVolumeSource{
				Driver:   "secrets-store.csi.k8s.io",
				ReadOnly: structures.PtrToBool(true),
				FSType:   structures.PtrToString("nfs"),
				VolumeAttributes: map[string]string{
					"secretProviderClass": "azure-keyvault",
				},
				NodePublishSecretRef: &v1.LocalObjectReference{
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
			Input: &v1.CSIVolumeSource{
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
