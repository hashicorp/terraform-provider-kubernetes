package kubernetes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
)

var (
	testNodeName = "test-nodename"
)

func TestFlattenEndpointsAddresses(t *testing.T) {

	cases := []struct {
		Input          []api.EndpointAddress
		ExpectedOutput *schema.Set
	}{
		{
			[]api.EndpointAddress{{
				Hostname: "any.hostname.io",
				IP:       "10.0.0.4",
				NodeName: &testNodeName,
			}},
			schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
				"hostname":  "any.hostname.io",
				"ip":        "10.0.0.4",
				"node_name": testNodeName,
			}}),
		},
		{
			[]api.EndpointAddress{{}},
			schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
				"ip": "",
			}}),
		},
		{
			[]api.EndpointAddress{},
			schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{}),
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
		ExpectedOutput *schema.Set
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
			schema.NewSet(hashEndpointsSubset(), []interface{}{map[string]interface{}{
				"address": schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
					"hostname":  "any.hostname.io",
					"ip":        "10.0.0.4",
					"node_name": testNodeName,
				}}),
				"not_ready_address": schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
					"hostname": "notready.hostname.io",
					"ip":       "10.0.0.5",
				}}),
				"port": schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
					"name":     "transport",
					"port":     8889,
					"protocol": "UDP",
				}}),
			}}),
		},
		{
			[]api.EndpointSubset{{}},
			schema.NewSet(hashEndpointsSubset(), []interface{}{map[string]interface{}{}}),
		},
		{
			[]api.EndpointSubset{},
			schema.NewSet(hashEndpointsSubset(), []interface{}{}),
		},
	}

	for _, tc := range cases {
		output := flattenEndpointsSubsets(tc.Input)

		// FIXME: not sure why this is required here but not in other flatteners tests
		output.F = nil
		tc.ExpectedOutput.F = nil
		if output.Len() > 0 {
			if output.List()[0].(map[string]interface{})["address"] != nil {
				output.List()[0].(map[string]interface{})["address"].(*schema.Set).F = nil
				tc.ExpectedOutput.List()[0].(map[string]interface{})["address"].(*schema.Set).F = nil
			}
			if output.List()[0].(map[string]interface{})["not_ready_address"] != nil {
				output.List()[0].(map[string]interface{})["not_ready_address"].(*schema.Set).F = nil
				tc.ExpectedOutput.List()[0].(map[string]interface{})["not_ready_address"].(*schema.Set).F = nil
			}
			if output.List()[0].(map[string]interface{})["port"] != nil {
				output.List()[0].(map[string]interface{})["port"].(*schema.Set).F = nil
				tc.ExpectedOutput.List()[0].(map[string]interface{})["port"].(*schema.Set).F = nil
			}
		}
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandEndpointsAddresses(t *testing.T) {

	cases := []struct {
		Input          *schema.Set
		ExpectedOutput []api.EndpointAddress
	}{
		{
			schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
				"hostname":  "any.hostname.io",
				"ip":        "10.0.0.4",
				"node_name": testNodeName,
			}}),
			[]api.EndpointAddress{{
				Hostname: "any.hostname.io",
				IP:       "10.0.0.4",
				NodeName: &testNodeName,
			}},
		},
		{
			schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{}}),
			[]api.EndpointAddress{{}},
		},
		{
			schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{}),
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
		Input          *schema.Set
		ExpectedOutput []api.EndpointSubset
	}{
		{
			schema.NewSet(hashEndpointsSubset(), []interface{}{map[string]interface{}{
				"address": schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
					"hostname":  "any.hostname.io",
					"ip":        "10.0.0.4",
					"node_name": testNodeName,
				}}),
				"not_ready_address": schema.NewSet(hashEndpointsSubsetAddress(), []interface{}{map[string]interface{}{
					"hostname": "notready.hostname.io",
					"ip":       "10.0.0.5",
				}}),
				"port": schema.NewSet(hashEndpointsSubsetPort(), []interface{}{map[string]interface{}{
					"name":     "transport",
					"port":     8889,
					"protocol": "UDP",
				}}),
			}}),
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
			schema.NewSet(hashEndpointsSubset(), []interface{}{map[string]interface{}{}}),
			[]api.EndpointSubset{{}},
		},
		{
			schema.NewSet(hashEndpointsSubset(), []interface{}{}),
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
