package kubernetes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/core/v1"
)

var (
	testNodeName = "test-nodename"
)

func TestFlattenEndpointsAddresses(t *testing.T) {

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
		output := flattenEndpointsAddresses(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenEndpointsPorts(t *testing.T) {

	cases := []struct {
		Input          []api.EndpointPort
		ExpectedOutput *schema.Set
	}{
		{
			[]api.EndpointPort{{
				Name:     "transport",
				Port:     80,
				Protocol: api.ProtocolTCP,
			}},
			schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
				"name":     "transport",
				"port":     80,
				"protocol": "TCP",
			}}),
		},
		{
			[]api.EndpointPort{{
				Port:     443,
				Protocol: api.ProtocolUDP,
			}},
			schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
				"port":     443,
				"protocol": "UDP",
			}}),
		},
		{
			[]api.EndpointPort{{}},
			schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
				"port":     0,
				"protocol": "",
			}}),
		},
		{
			[]api.EndpointPort{},
			schema.NewSet(hashEndpointsSubsetPort(), []interface{}{}),
		},
	}

	for _, tc := range cases {
		output := flattenEndpointsPorts(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenEndpointsSubsets(t *testing.T) {

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
					"address": []interface{}{
						map[string]interface{}{
							"hostname":  "any.hostname.io",
							"ip":        "10.0.0.4",
							"node_name": testNodeName,
						},
					},
					"not_ready_address": []interface{}{
						map[string]interface{}{
							"hostname": "notready.hostname.io",
							"ip":       "10.0.0.5",
						},
					},
					"port": schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
						"name":     "transport",
						"port":     8889,
						"protocol": "UDP",
					}}),
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
		output := flattenEndpointsSubsets(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandEndpointsAddresses(t *testing.T) {

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
		output := expandEndpointsAddresses(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandEndpointsPorts(t *testing.T) {

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
		output := expandEndpointsPorts(schema.NewSet(hashEndpointsSubsetPort(), tc.Input))
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandEndpointsSubsets(t *testing.T) {

	cases := []struct {
		Input          []interface{}
		ExpectedOutput []api.EndpointSubset
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"address": []interface{}{
						map[string]interface{}{
							"hostname":  "any.hostname.io",
							"ip":        "10.0.0.4",
							"node_name": testNodeName,
						},
					},
					"not_ready_address": []interface{}{
						map[string]interface{}{
							"hostname": "notready.hostname.io",
							"ip":       "10.0.0.5",
						},
					},
					"port": schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
						"name":     "transport",
						"port":     8889,
						"protocol": "UDP",
					}}),
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
		output := expandEndpointsSubsets(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}
