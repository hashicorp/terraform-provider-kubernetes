// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestFlattenServiceSpecLoadBalancerIP(t *testing.T) {
	cases := []struct {
		Input    v1.ServiceSpec
		Expected string
	}{
		{
			v1.ServiceSpec{Type: v1.ServiceTypeLoadBalancer, LoadBalancerIP: "192.168.1.1"},
			"192.168.1.1",
		},
		{
			v1.ServiceSpec{Type: v1.ServiceTypeLoadBalancer, LoadBalancerIP: ""},
			"",
		},
		{
			v1.ServiceSpec{Type: v1.ServiceTypeLoadBalancer, LoadBalancerIP: "None"},
			"None",
		},
	}

	for _, tc := range cases {
		output := flattenServiceSpec(tc.Input)
		m := output[0].(map[string]interface{})
		got, ok := m["load_balancer_ip"]
		if !ok {
			t.Fatalf("load_balancer_ip key missing from flattened output for LoadBalancerIP=%q", tc.Input.LoadBalancerIP)
		}
		if got != tc.Expected {
			t.Fatalf("Expected load_balancer_ip=%q, got %q", tc.Expected, got)
		}
	}
}

func TestExpandServiceSpecLoadBalancerIP(t *testing.T) {
	cases := []struct {
		Input    []interface{}
		Expected v1.ServiceSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"load_balancer_ip": "192.168.1.1",
				},
			},
			v1.ServiceSpec{
				LoadBalancerIP: "192.168.1.1",
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"load_balancer_ip": "",
				},
			},
			v1.ServiceSpec{
				LoadBalancerIP: "",
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"load_balancer_ip": "None",
				},
			},
			v1.ServiceSpec{
				LoadBalancerIP: "None",
			},
		},
	}

	for _, tc := range cases {
		output := expandServiceSpec(tc.Input)
		if output.LoadBalancerIP != tc.Expected.LoadBalancerIP {
			t.Fatalf("Expected LoadBalancerIP=%q, got %q", tc.Expected.LoadBalancerIP, output.LoadBalancerIP)
		}
	}
}
