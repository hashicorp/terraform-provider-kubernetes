// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBatchFrameworkFlattenDoesNotMutateAPI(t *testing.T) {
	spec := batchv1.JobSpec{
		BackoffLimit: ptr.To(int32(6)),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				"batch.kubernetes.io/job-name": "test-job", "configured": "value",
			}},
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				Containers: []corev1.Container{{
					Name: "test", Image: "busybox:1.36",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Mi")},
					},
				}},
			},
		},
	}
	before := spec.DeepCopy()
	job, err := FlattenJobV1Spec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(spec, *before) {
		t.Fatal("Framework flattener changed the API Job spec")
	}
	template := job[0].(map[string]interface{})["template"].([]interface{})[0].(map[string]interface{})
	metadata := template["metadata"].([]interface{})[0].(map[string]interface{})
	labels := metadata["labels"].(map[string]string)
	if _, exists := labels["batch.kubernetes.io/job-name"]; exists || labels["configured"] != "value" {
		t.Fatalf("unexpected filtered labels: %#v", labels)
	}
	cron := batchv1.CronJobSpec{Schedule: "0 * * * *", JobTemplate: batchv1.JobTemplateSpec{Spec: spec}}
	cronBefore := cron.DeepCopy()
	if _, err := FlattenCronJobSpecV1(cron); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cron, *cronBefore) {
		t.Fatal("Framework flattener changed the API CronJob spec")
	}
}
