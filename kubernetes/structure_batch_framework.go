// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import batchv1 "k8s.io/api/batch/v1"

// ExpandJobV1Spec shares the workload conversion with Framework without constructing SDK state.
func ExpandJobV1Spec(spec []interface{}) (batchv1.JobSpec, error) {
	return expandJobV1Spec(spec)
}

// FlattenJobV1Spec only needs the API object; the legacy state arguments are unused.
func FlattenJobV1Spec(spec batchv1.JobSpec) ([]interface{}, error) {
	return flattenJobV1Spec(*spec.DeepCopy(), nil, nil)
}

func ExpandCronJobSpecV1(spec []interface{}) (batchv1.CronJobSpec, error) {
	return expandCronJobSpecV1(spec)
}

func FlattenCronJobSpecV1(spec batchv1.CronJobSpec) ([]interface{}, error) {
	return flattenCronJobSpecV1(*spec.DeepCopy(), nil, nil)
}
