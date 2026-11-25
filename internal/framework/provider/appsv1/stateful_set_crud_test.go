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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
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

// Removing the update_strategy block returns the StatefulSet to the API default.
func TestStatefulSetPatchSpecsResetsRemovedUpdateStrategy(t *testing.T) {
	for name, live := range map[string]appsv1.StatefulSetUpdateStrategy{
		"OnDelete":  {Type: appsv1.OnDeleteStatefulSetStrategyType},
		"partition": {Type: appsv1.RollingUpdateStatefulSetStrategyType, RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: ptr.To(int32(2))}},
		"defaulted": rollingUpdateSpec("1").UpdateStrategy,
	} {
		t.Run(name, func(t *testing.T) {
			spec := appsv1.StatefulSetSpec{UpdateStrategy: live}
			_, to := statefulSetPatchSpecs(spec, appsv1.StatefulSetSpec{}, spec)
			if to.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType || to.UpdateStrategy.RollingUpdate != nil {
				t.Errorf("update strategy = %+v, want RollingUpdate with no rolling update", to.UpdateStrategy)
			}
		})
	}
}

func rollingUpdateSpec(maxUnavailable string) appsv1.StatefulSetSpec {
	rolling := &appsv1.RollingUpdateStatefulSetStrategy{}
	if maxUnavailable != "" {
		rolling.MaxUnavailable = ptr.To(intstr.Parse(maxUnavailable))
	}
	return appsv1.StatefulSetSpec{UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType, RollingUpdate: rolling}}
}

func TestStatefulSetPatchSpecsMaxUnavailable(t *testing.T) {
	for name, tc := range map[string]struct{ original, desired, live, want string }{
		"server default kept": {live: "1", want: "1"}, // not configured, so not managed
		"configured":          {desired: "25%", live: "1", want: "25%"},
		"removed":             {original: "2", live: "2", want: "<nil>"}, // the server then applies its default
	} {
		if _, to := statefulSetPatchSpecs(rollingUpdateSpec(tc.original), rollingUpdateSpec(tc.desired), rollingUpdateSpec(tc.live)); to.UpdateStrategy.RollingUpdate.MaxUnavailable.String() != tc.want {
			t.Errorf("%s: max unavailable = %s, want %s", name, to.UpdateStrategy.RollingUpdate.MaxUnavailable, tc.want)
		}
	}
}

// A server that drops ordinals leaves the pods numbered from zero.
func TestStatefulSetFeaturePreservation(t *testing.T) {
	ordinals := func(start int32) appsv1.StatefulSetSpec {
		return appsv1.StatefulSetSpec{Ordinals: &appsv1.StatefulSetOrdinals{Start: start}}
	}
	class := func(name *string) appsv1.StatefulSetSpec {
		return appsv1.StatefulSetSpec{VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{Spec: corev1.PersistentVolumeClaimSpec{VolumeAttributesClassName: name}}}}
	}
	for name, tc := range map[string]struct {
		want, got appsv1.StatefulSetSpec
		preflight bool
		dropped   bool
	}{
		"unset":           {},
		"kept":            {want: ordinals(5), got: ordinals(5), preflight: true},
		"dropped":         {want: ordinals(5), preflight: true, dropped: true},
		"changed":         {want: ordinals(5), got: ordinals(1), preflight: true, dropped: true},
		"zero is default": {want: ordinals(0)},
		"max kept":        {want: rollingUpdateSpec("2"), got: rollingUpdateSpec("2"), preflight: true},
		"max dropped":     {want: rollingUpdateSpec("2"), got: rollingUpdateSpec(""), preflight: true, dropped: true},
		"max defaulted":   {want: rollingUpdateSpec(""), got: rollingUpdateSpec("1")},
		"class kept":      {want: class(ptr.To("gold")), got: class(ptr.To("gold")), preflight: true},
		"class dropped":   {want: class(ptr.To("gold")), got: class(nil), preflight: true, dropped: true},
		"class unset":     {want: class(nil), got: class(ptr.To("gold"))},
	} {
		t.Run(name, func(t *testing.T) {
			want, got := tc.want, tc.got
			if hasStatefulSetFeatures(&want) != tc.preflight {
				t.Errorf("preflight = %t, want %t", !tc.preflight, tc.preflight)
			}
			if err := checkStatefulSetFeatures(&want, &got); (err != nil) != tc.dropped {
				t.Errorf("error = %v, want error = %t", err, tc.dropped)
			}
		})
	}
}
