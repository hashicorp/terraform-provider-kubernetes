package kubernetes

import (
	"reflect"
	"testing"

	"k8s.io/api/networking/v1"
)

// Test Flatteners
func TestFlattenIngressRule(t *testing.T) {
	pathType := v1.PathTypeExact
	r := v1.HTTPIngressRuleValue{
		Paths: []v1.HTTPIngressPath{
			{
				Path:     "/foo/bar",
				PathType: &pathType,
				Backend: v1.IngressBackend{
					Service: &v1.IngressServiceBackend{
						Name: "foo",
						Port: v1.ServiceBackendPort{
							Number: 1234,
						},
					},
				},
			},
		},
	}

	in := []v1.IngressRule{
		{
			Host: "the-app-name.staging.live.domain-replaced.tld",
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: (*v1.HTTPIngressRuleValue)(nil),
			},
		},
		{
			Host: "",
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: (*v1.HTTPIngressRuleValue)(&r),
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
							"path":      "/foo/bar",
							"path_type": "Exact",
							"backend": []interface{}{
								map[string]interface{}{
									"service": []interface{}{
										map[string]interface{}{
											"name": "foo",
											"port": []interface{}{
												map[string]interface{}{
													"port_number": "1234",
												},
											},
										},
									},
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
