package kubernetes

import (
	"reflect"
	"testing"

	api "k8s.io/api/core/v1"
)

var (
	testNodeName = "test-nodename"
)

func TestFlattenEndpointAddresses(t *testing.T) {

	cases := []struct {
		Input          []api.EndpointAddress
		ExpectedOutput []interface{}
	}{
		{
			[]api.EndpointAddress{{
				Hostname: "any.hostname.io",
				IP:       "10.0.0.4",
				NodeName: &testNodeName,
			}},
			[]interface{}{
				map[string]interface{}{
					"hostname":  "any.hostname.io",
					"ip":        "10.0.0.4",
					"node_name": testNodeName,
				},
			},
		},
		{
			[]api.EndpointAddress{{}},
			[]interface{}{map[string]interface{}{
				"ip": "",
			}},
		},
		{
			[]api.EndpointAddress{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenEndpointAddresses(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenEndpointPorts(t *testing.T) {

	cases := []struct {
		Input          []api.EndpointPort
		ExpectedOutput []interface{}
	}{
		{
			[]api.EndpointPort{{
				Name:     "transport",
				Port:     80,
				Protocol: api.ProtocolTCP,
			}},
			[]interface{}{
				map[string]interface{}{
					"name":     "transport",
					"port":     80,
					"protocol": "TCP",
				},
			},
		},
		{
			[]api.EndpointPort{{
				Port:     443,
				Protocol: api.ProtocolUDP,
			}},
			[]interface{}{
				map[string]interface{}{
					"port":     443,
					"protocol": "UDP",
				},
			},
		},
		{
			[]api.EndpointPort{{}},
			[]interface{}{map[string]interface{}{
				"port":     0,
				"protocol": "",
			}},
		},
		{
			[]api.EndpointPort{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenEndpointPorts(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenEndpointSubsets(t *testing.T) {

	cases := []struct {
		Input          []api.EndpointSubset
		ExpectedOutput []interface{}
	}{
		{
			[]api.EndpointSubset{
				{
					Addresses: []api.EndpointAddress{
						{
							Hostname: "any.hostname.io",
							IP:       "10.0.0.4",
							NodeName: &testNodeName,
						},
					},
					NotReadyAddresses: []api.EndpointAddress{
						{
							Hostname: "notready.hostname.io",
							IP:       "10.0.0.5",
						},
					},
					Ports: []api.EndpointPort{
						{
							Name:     "transport",
							Port:     8889,
							Protocol: api.ProtocolUDP,
						},
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"addresses": []interface{}{
						map[string]interface{}{
							"hostname":  "any.hostname.io",
							"ip":        "10.0.0.4",
							"node_name": testNodeName,
						},
					},
					"not_ready_addresses": []interface{}{
						map[string]interface{}{
							"hostname": "notready.hostname.io",
							"ip":       "10.0.0.5",
						},
					},
					"ports": []interface{}{
						map[string]interface{}{
							"name":     "transport",
							"port":     8889,
							"protocol": "UDP",
						},
					},
				},
			},
		},
		{
			[]api.EndpointSubset{{}},
			[]interface{}{map[string]interface{}{}},
		},
		{
			[]api.EndpointSubset{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenEndpointSubsets(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandEndpointAddresses(t *testing.T) {

	cases := []struct {
		Input          []interface{}
		ExpectedOutput []api.EndpointAddress
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"hostname":  "any.hostname.io",
					"ip":        "10.0.0.4",
					"node_name": testNodeName,
				},
			},
			[]api.EndpointAddress{{
				Hostname: "any.hostname.io",
				IP:       "10.0.0.4",
				NodeName: &testNodeName,
			}},
		},
		{
			[]interface{}{map[string]interface{}{}},
			[]api.EndpointAddress{{}},
		},
		{
			[]interface{}{},
			[]api.EndpointAddress{},
		},
	}

	for _, tc := range cases {
		output := expandEndpointAddresses(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandEndpointPorts(t *testing.T) {

	cases := []struct {
		Input          []interface{}
		ExpectedOutput []api.EndpointPort
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"name":     "transport",
					"port":     80,
					"protocol": "TCP",
				},
			},
			[]api.EndpointPort{{
				Name:     "transport",
				Port:     80,
				Protocol: api.ProtocolTCP,
			}},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"port":     443,
					"protocol": "UDP",
				},
			},
			[]api.EndpointPort{{
				Port:     443,
				Protocol: api.ProtocolUDP,
			}},
		},
		{
			[]interface{}{map[string]interface{}{}},
			[]api.EndpointPort{{}},
		},
		{
			[]interface{}{},
			[]api.EndpointPort{},
		},
	}

	for _, tc := range cases {
		output := expandEndpointPorts(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandEndpointSubsets(t *testing.T) {

	cases := []struct {
		Input          []interface{}
		ExpectedOutput []api.EndpointSubset
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"addresses": []interface{}{
						map[string]interface{}{
							"hostname":  "any.hostname.io",
							"ip":        "10.0.0.4",
							"node_name": testNodeName,
						},
					},
					"not_ready_addresses": []interface{}{
						map[string]interface{}{
							"hostname": "notready.hostname.io",
							"ip":       "10.0.0.5",
						},
					},
					"ports": []interface{}{
						map[string]interface{}{
							"name":     "transport",
							"port":     8889,
							"protocol": "UDP",
						},
					},
				},
			},
			[]api.EndpointSubset{
				{
					Addresses: []api.EndpointAddress{
						{
							Hostname: "any.hostname.io",
							IP:       "10.0.0.4",
							NodeName: &testNodeName,
						},
					},
					NotReadyAddresses: []api.EndpointAddress{
						{
							Hostname: "notready.hostname.io",
							IP:       "10.0.0.5",
						},
					},
					Ports: []api.EndpointPort{
						{
							Name:     "transport",
							Port:     8889,
							Protocol: api.ProtocolUDP,
						},
					},
				},
			},
		},
		{
			[]interface{}{map[string]interface{}{}},
			[]api.EndpointSubset{{}},
		},
		{
			[]interface{}{},
			[]api.EndpointSubset{},
		},
	}

	for _, tc := range cases {
		output := expandEndpointSubsets(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
