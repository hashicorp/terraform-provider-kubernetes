// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"reflect"
	"testing"

	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Test Flatteners
func TestFlattenIngressRule(t *testing.T) {
	r := v1beta1.HTTPIngressRuleValue{
		Paths: []v1beta1.HTTPIngressPath{
			{
				Path: "/foo/bar",
				Backend: v1beta1.IngressBackend{
					ServiceName: "foo",
					ServicePort: intstr.FromInt(1234),
				},
			},
		},
	}

	in := []v1beta1.IngressRule{
		{
			Host: "the-app-name.staging.live.domain-replaced.tld",
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: (*v1beta1.HTTPIngressRuleValue)(nil),
			},
		},
		{
			Host: "",
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: (*v1beta1.HTTPIngressRuleValue)(&r),
			},
		},
	}
	out := []interface{}{
		map[string]interface{}{
			"host": "the-app-name.staging.live.domain-replaced.tld",
			"http": []interface{}{},
		},
		map[string]interface{}{
			"host": "",
			"http": []interface{}{
				map[string]interface{}{
					"path": []interface{}{
						map[string]interface{}{
							"path": "/foo/bar",
							"backend": []interface{}{
								map[string]interface{}{
									"service_name": "foo",
									"service_port": "1234",
								},
							},
						},
					},
				},
			},
		},
	}

	flatRules := flattenIngressRule(in)

	if len(flatRules) < len(out) {
		t.Error("Failed to flatten ingress rules")
	}

	for i, v := range flatRules {
		control := v.(map[string]interface{})
		sample := out[i]

		if !reflect.DeepEqual(control, sample) {
			t.Errorf("Unexpected result:\n\tWant:%s\n\tGot:%s\n", control, sample)
		}
	}
}
