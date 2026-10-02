// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestResolveUnconfigured(t *testing.T) {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"path": tftypes.String, "port": tftypes.String, "labels": tftypes.Map{ElementType: tftypes.String},
	}}
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	null := tftypes.NewValue(tftypes.String, nil)
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	labels := func(kv ...string) tftypes.Value {
		values := map[string]tftypes.Value{}
		for i := 0; i < len(kv); i += 2 {
			values[kv[i]] = str(kv[i+1])
		}
		return tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, values)
	}
	object := func(path, port, labels tftypes.Value) tftypes.Value {
		return tftypes.NewValue(typ, map[string]tftypes.Value{"path": path, "port": port, "labels": labels})
	}
	noLabels := tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil)
	state := object(str("/"), str("80"), labels("a", "b"))
	older := object(str("/"), null, labels("a", "b"))
	for name, tc := range map[string]struct {
		config, plan tftypes.Value
		want         tftypes.Value
		ok           bool
		state        *tftypes.Value
	}{
		"unset scalar is removed": {
			object(null, str("80"), labels("a", "b")), object(null, str("80"), labels("a", "b")),
			object(null, str("80"), labels("a", "b")), true, nil,
		},
		"unset scalar keeps its planned default": {
			object(null, str("80"), labels("a", "b")), object(str(""), str("80"), labels("a", "b")),
			object(str(""), str("80"), labels("a", "b")), true, nil,
		},
		"empty string defers to the prior value": {
			object(str(""), str("80"), labels("a", "b")), object(str(""), str("80"), labels("a", "b")),
			state, true, nil,
		},
		"configured change is kept": {
			object(str("/"), str("8080"), labels("a", "b")), object(str("/"), str("8080"), labels("a", "b")),
			object(str("/"), str("8080"), labels("a", "b")), true, nil,
		},
		"unset collection is removed": {
			object(str("/"), str("80"), noLabels), object(str("/"), str("80"), noLabels),
			object(str("/"), str("80"), noLabels), true, nil,
		},
		"zero default of a field older state lacks": {
			object(null, null, labels("a", "b")), object(str(""), str(""), labels("a", "b")),
			object(str(""), null, labels("a", "b")), true, &older,
		},
		"unknown configuration cannot be resolved": {
			object(null, unknown, labels("a", "b")), object(str(""), unknown, labels("a", "b")),
			tftypes.Value{}, false, nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			prior := state
			if tc.state != nil {
				prior = *tc.state
			}
			got, ok := resolveUnconfigured(tc.config, tc.plan, prior)
			if ok != tc.ok || (ok && !got.Equal(tc.want)) {
				t.Errorf("got %s, %t; want %s, %t", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestPayloadsEqual(t *testing.T) {
	for name, tc := range map[string]struct {
		a, b corev1.PodSpec
		want bool
	}{
		"empty and absent security context": {
			corev1.PodSpec{SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(false), SupplementalGroups: []int64{}}},
			corev1.PodSpec{}, true,
		},
		"empty and absent maps": {
			corev1.PodSpec{NodeSelector: map[string]string{}}, corev1.PodSpec{}, true,
		},
		"respelled quantity": {
			corev1.PodSpec{Containers: []corev1.Container{{Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": resource.MustParse("0.5")}}}}},
			corev1.PodSpec{Containers: []corev1.Container{{Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": resource.MustParse("500m")}}}}},
			true,
		},
		"changed value": {
			corev1.PodSpec{SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(true)}}, corev1.PodSpec{}, false,
		},
		"element order": {
			corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}, {Name: "b"}}},
			corev1.PodSpec{Containers: []corev1.Container{{Name: "b"}, {Name: "a"}}},
			false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := payloadsEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("payloadsEqual = %t, want %t", got, tc.want)
			}
		})
	}
}
