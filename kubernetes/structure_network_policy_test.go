// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	api "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"

	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	protoTCP      = api.ProtocolTCP
	protoUDP      = api.ProtocolUDP
	portName      = intstr.FromString("http")
	portNumerical = intstr.FromInt(8125)
)

func TestFlattenNetworkPolicyIngressPorts(t *testing.T) {

	cases := []struct {
		Input          []v1.NetworkPolicyPort
		ExpectedOutput []interface{}
	}{
		{
			[]v1.NetworkPolicyPort{{
				Port:     &portName,
				Protocol: &protoTCP,
			}},
			[]interface{}{
				map[string]interface{}{
					"port":     "http",
					"protocol": "TCP",
				},
			},
		},
		{
			[]v1.NetworkPolicyPort{{
				Port: &portName,
			}},
			[]interface{}{
				map[string]interface{}{
					"port": "http",
				},
			},
		},
		{
			[]v1.NetworkPolicyPort{{
				Port:     &portNumerical,
				Protocol: &protoUDP,
			}},
			[]interface{}{
				map[string]interface{}{
					"port":     "8125",
					"protocol": "UDP",
				},
			},
		},
		{
			[]v1.NetworkPolicyPort{{}},
			[]interface{}{map[string]interface{}{}},
		},
		{
			[]v1.NetworkPolicyPort{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenNetworkPolicyPorts(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandNetworkPolicyIngressPorts(t *testing.T) {

	cases := []struct {
		Input          []interface{}
		ExpectedOutput []v1.NetworkPolicyPort
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"port":     "http",
					"protocol": "TCP",
				},
			},
			[]v1.NetworkPolicyPort{{
				Port:     &portName,
				Protocol: &protoTCP,
			}},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"port": "http",
				},
			},
			[]v1.NetworkPolicyPort{{
				Port: &portName,
			}},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"port":     "8125",
					"protocol": "UDP",
				},
			},
			[]v1.NetworkPolicyPort{{
				Port:     &portNumerical,
				Protocol: &protoUDP,
			}},
		},
		{
			[]interface{}{map[string]interface{}{}},
			[]v1.NetworkPolicyPort{{}},
		},
		{
			[]interface{}{},
			[]v1.NetworkPolicyPort{},
		},
	}

	for _, tc := range cases {
		output, _ := expandNetworkPolicyPorts(tc.Input)
		if !reflect.DeepEqual(output, &tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
