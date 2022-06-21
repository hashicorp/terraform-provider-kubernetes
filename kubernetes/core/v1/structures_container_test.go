package v1

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"
	v1 "k8s.io/api/core/v1"
)

func TestFlattenSecretKeyRef(t *testing.T) {
	cases := []struct {
		Input          *v1.SecretKeySelector
		ExpectedOutput []interface{}
	}{
		{
			&v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "Secret1",
				},
				Key:      "key1",
				Optional: structures.PtrToBool(true),
			},
			[]interface{}{
				map[string]interface{}{
					"key":      "key1",
					"name":     "Secret1",
					"optional": true,
				},
			},
		},
		{
			&v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "Secret2",
				},
				Key: "key2",
			},
			[]interface{}{
				map[string]interface{}{
					"key":  "key2",
					"name": "Secret2",
				},
			},
		},
		{
			&v1.SecretKeySelector{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenSecretKeyRef(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandSecretKeyRef(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *v1.SecretKeySelector
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"key":      "key1",
					"name":     "Secret1",
					"optional": true,
				},
			},
			&v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "Secret1",
				},
				Key:      "key1",
				Optional: structures.PtrToBool(true),
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"key":  "key2",
					"name": "Secret2",
				},
			},
			&v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "Secret2",
				},
				Key: "key2",
			},
		},
		{
			[]interface{}{},
			&v1.SecretKeySelector{},
		},
	}

	for _, tc := range cases {
		output, err := expandSecretKeyRef(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenConfigMapKeyRef(t *testing.T) {
	cases := []struct {
		Input          *v1.ConfigMapKeySelector
		ExpectedOutput []interface{}
	}{
		{
			&v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap1",
				},
				Key:      "key1",
				Optional: structures.PtrToBool(true),
			},
			[]interface{}{
				map[string]interface{}{
					"key":      "key1",
					"name":     "configmap1",
					"optional": true,
				},
			},
		},
		{
			&v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap2",
				},
				Key: "key2",
			},
			[]interface{}{
				map[string]interface{}{
					"key":  "key2",
					"name": "configmap2",
				},
			},
		},
		{
			&v1.ConfigMapKeySelector{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenConfigMapKeyRef(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandConfigMapKeyRef(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *v1.ConfigMapKeySelector
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"key":      "key1",
					"name":     "configmap1",
					"optional": true,
				},
			},
			&v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap1",
				},
				Key:      "key1",
				Optional: structures.PtrToBool(true),
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"key":  "key2",
					"name": "configmap2",
				},
			},
			&v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "configmap2",
				},
				Key: "key2",
			},
		},
		{
			[]interface{}{},
			&v1.ConfigMapKeySelector{},
		},
	}

	for _, tc := range cases {
		output, err := expandConfigMapKeyRef(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandContainerEnv(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput []v1.EnvVar
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"name":  "PGUSER",
					"value": "postgres",
				},
				map[string]interface{}{
					"name":  "PGHOST",
					"value": "localhost",
				},
			},
			[]v1.EnvVar{
				{
					Name:  "PGUSER",
					Value: "postgres",
				},
				{
					Name:  "PGHOST",
					Value: "localhost",
				},
			},
		},
		{
			[]interface{}{nil},
			[]v1.EnvVar{},
		},
	}

	for _, tc := range cases {
		output, err := expandContainerEnv(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
