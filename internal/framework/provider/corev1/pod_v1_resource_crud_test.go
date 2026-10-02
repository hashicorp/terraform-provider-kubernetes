// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodV1UpdatePatch(t *testing.T) {
	spec := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "image:1"}}}
	for _, test := range []struct {
		name   string
		prior  metav1.ObjectMeta
		plan   metav1.ObjectMeta
		live   metav1.ObjectMeta
		expect map[string]interface{}
	}{
		{
			name: "state-only empty maps",
			plan: metav1.ObjectMeta{Labels: map[string]string{}, Annotations: map[string]string{}},
			live: metav1.ObjectMeta{Labels: map[string]string{"controller": "keep"}},
		},
		{
			name: "first managed key retains external metadata",
			plan: metav1.ObjectMeta{Labels: map[string]string{"managed": "new"}},
			live: metav1.ObjectMeta{Labels: map[string]string{"controller": "keep"}},
			expect: map[string]interface{}{
				"metadata": map[string]interface{}{"labels": map[string]interface{}{"managed": "new"}},
			},
		},
		{
			name:  "remove only managed keys",
			prior: metav1.ObjectMeta{Annotations: map[string]string{"managed/x~y": "old"}},
			live:  metav1.ObjectMeta{Annotations: map[string]string{"managed/x~y": "old", "controller": "keep"}},
			expect: map[string]interface{}{
				"metadata": map[string]interface{}{"annotations": map[string]interface{}{"managed/x~y": nil}},
			},
		},
		{
			name:  "already converged metadata",
			prior: metav1.ObjectMeta{Labels: map[string]string{"managed": "old"}},
			plan:  metav1.ObjectMeta{Labels: map[string]string{"managed": "new"}},
			live:  metav1.ObjectMeta{Labels: map[string]string{"managed": "new", "controller": "keep"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := podV1UpdatePatch(test.prior, test.plan, &spec, &spec, &corev1.Pod{
				ObjectMeta: test.live,
				Spec:       spec,
			})
			if test.expect == nil {
				test.expect = map[string]interface{}{}
			}
			if !reflect.DeepEqual(actual, test.expect) {
				t.Fatalf("patch = %#v, want %#v", actual, test.expect)
			}
		})
	}
	t.Run("active deadline edit", func(t *testing.T) {
		seconds := int64(120)
		planned := spec.DeepCopy()
		planned.ActiveDeadlineSeconds = &seconds
		patch := podV1UpdatePatch(metav1.ObjectMeta{}, metav1.ObjectMeta{}, &spec, planned, &corev1.Pod{Spec: spec})
		data, err := json.Marshal(patch)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"spec":{"activeDeadlineSeconds":120}}` {
			t.Fatalf("unexpected patch: %s", data)
		}
	})
}
