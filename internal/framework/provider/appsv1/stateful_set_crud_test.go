// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"testing"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// State written by SDKv2 holds null where the configuration holds {}; the API
// sees no difference, so Update must not patch the spec.
func TestStatefulSetPatchSpecsSkipsNoOpChanges(t *testing.T) {
	one := int32(1)
	base := appsv1.StatefulSetSpec{
		Replicas:    &one,
		ServiceName: "s",
		Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "s"}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "s"}},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
		},
		UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
	}
	for _, tc := range []struct {
		name   string
		change func(*appsv1.StatefulSetSpec)
		patch  bool
	}{
		{"null to empty maps and lists", func(s *appsv1.StatefulSetSpec) {
			s.Template.Annotations = map[string]string{}
			s.Template.Spec.NodeSelector = map[string]string{}
			s.Template.Spec.Containers[0].Args = []string{}
		}, false},
		{"template annotation added", func(s *appsv1.StatefulSetSpec) {
			s.Template.Annotations = map[string]string{"a": "b"}
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired := *base.DeepCopy()
			tc.change(&desired)
			object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&appsv1.StatefulSet{Spec: base})
			if err != nil {
				t.Fatal(err)
			}
			from, to := statefulSetPatchSpecs(*base.DeepCopy(), desired, *base.DeepCopy())
			ops, err := common.StrategicMergeSpecOps(&unstructured.Unstructured{Object: object}, from, to, appsv1.StatefulSet{})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(ops) > 0; got != tc.patch {
				t.Errorf("patch = %t, want %t", got, tc.patch)
			}
		})
	}
}
