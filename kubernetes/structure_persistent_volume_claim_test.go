package kubernetes

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	corev1 "k8s.io/api/core/v1"
)

func TestExpandPersistentVolumeClaimSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.PersistentVolumeClaimSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"resources":          []interface{}{},
					"access_modes":       &schema.Set{},
					"storage_class_name": "",
				},
			},
			&corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{},
				Selector:         nil,
				Resources:        corev1.ResourceRequirements{},
				VolumeName:       "",
				StorageClassName: ptrToString(""),
				VolumeMode:       nil,
				DataSource:       nil,
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"resources":          []interface{}{},
					"access_modes":       &schema.Set{},
					"storage_class_name": nil,
				},
			},
			&corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{},
				Selector:         nil,
				Resources:        corev1.ResourceRequirements{},
				VolumeName:       "",
				StorageClassName: nil,
				VolumeMode:       nil,
				DataSource:       nil,
			},
		},
	}

	for _, tc := range cases {
		output, err := expandPersistentVolumeClaimSpec(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
