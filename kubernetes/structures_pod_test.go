package kubernetes

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestFlattenTolerations(t *testing.T) {
	cases := []struct {
		Input          []v1.Toleration
		ExpectedOutput []interface{}
	}{
		{
			[]v1.Toleration{
				v1.Toleration{
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
				v1.Toleration{
					Key:      "node-role.kubernetes.io/other-worker",
					Operator: "Exists",
				},
				v1.Toleration{
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
				v1.Toleration{
					Effect:            "NoExecute",
					TolerationSeconds: ptrToInt64(120),
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
				&v1.Toleration{
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
				&v1.Toleration{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
				&v1.Toleration{
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
				&v1.Toleration{
					Effect:            "NoExecute",
					TolerationSeconds: ptrToInt64(120),
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
