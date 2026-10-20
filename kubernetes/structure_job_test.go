// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/utils/ptr"
)

func TestFlattenPodFailurePolicyOnExitCodes(t *testing.T) {
	cases := []struct {
		Name           string
		Input          *batchv1.PodFailurePolicyOnExitCodesRequirement
		ExpectedOutput []interface{}
	}{
		{
			// containerName is optional in the API; when it is omitted the
			// pointer is nil and must not be dereferenced.
			Name: "without container_name",
			Input: &batchv1.PodFailurePolicyOnExitCodesRequirement{
				Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
				Values:   []int32{3},
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"operator": batchv1.PodFailurePolicyOnExitCodesOpIn,
					"values":   []int{3},
				},
			},
		},
		{
			Name: "with empty container_name",
			Input: &batchv1.PodFailurePolicyOnExitCodesRequirement{
				ContainerName: ptr.To(""),
				Operator:      batchv1.PodFailurePolicyOnExitCodesOpNotIn,
				Values:        []int32{1, 2},
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"operator": batchv1.PodFailurePolicyOnExitCodesOpNotIn,
					"values":   []int{1, 2},
				},
			},
		},
		{
			Name: "with container_name",
			Input: &batchv1.PodFailurePolicyOnExitCodesRequirement{
				ContainerName: ptr.To("test"),
				Operator:      batchv1.PodFailurePolicyOnExitCodesOpIn,
				Values:        []int32{1, 2, 42},
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"container_name": "test",
					"operator":       batchv1.PodFailurePolicyOnExitCodesOpIn,
					"values":         []int{1, 2, 42},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			output := flattenPodFailurePolicyOnExitCodes(tc.Input)
			if !reflect.DeepEqual(output, tc.ExpectedOutput) {
				t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
					tc.ExpectedOutput, output)
			}
		})
	}
}

func TestFlattenJobV1SpecPodFailurePolicyWithoutContainerName(t *testing.T) {
	// Mirrors what the API server returns for a Job/CronJob whose
	// pod_failure_policy rule sets on_exit_codes without container_name.
	in := batchv1.JobSpec{
		PodFailurePolicy: &batchv1.PodFailurePolicy{
			Rules: []batchv1.PodFailurePolicyRule{
				{
					Action: batchv1.PodFailurePolicyActionFailJob,
					OnExitCodes: &batchv1.PodFailurePolicyOnExitCodesRequirement{
						Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
						Values:   []int32{3},
					},
				},
			},
		},
	}

	out, err := flattenJobV1Spec(in, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pfp := out[0].(map[string]interface{})["pod_failure_policy"].([]interface{})
	rule := pfp[0].(map[string]interface{})["rule"].([]interface{})[0].(map[string]interface{})
	onExitCodes := rule["on_exit_codes"].([]interface{})[0].(map[string]interface{})
	if _, ok := onExitCodes["container_name"]; ok {
		t.Fatalf("expected no container_name, got %#v", onExitCodes)
	}
	if !reflect.DeepEqual(onExitCodes["values"], []int{3}) {
		t.Fatalf("unexpected values: %#v", onExitCodes["values"])
	}
}
