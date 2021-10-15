package kubernetes

import (
	"reflect"
	"testing"

	networking "k8s.io/api/networking/v1"
)

// Test Flatteners
func TestFlattenIngressV1Rule(t *testing.T) {
	r := networking.HTTPIngressRuleValue{
		Paths: []networking.HTTPIngressPath{
			{
				Path: "/foo/bar",
				Backend: networking.IngressBackend{
					Service: &networking.IngressServiceBackend{
						Name: "foo",
						Port: networking.ServiceBackendPort{
							Number: 1234,
						},
					},
				},
			},
		},
	}

	in := []networking.IngressRule{
		{
			Host: "the-app-name.staging.live.domain-replaced.tld",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: (*networking.HTTPIngressRuleValue)(nil),
			},
		},
		{
			Host: "",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: (*networking.HTTPIngressRuleValue)(&r),
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

	flatRules := flattenIngressV1Rule(in)

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
