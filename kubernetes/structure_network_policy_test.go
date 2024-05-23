// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	protoTCP      = corev1.ProtocolTCP
	protoUDP      = corev1.ProtocolUDP
	portName      = intstr.FromString("http")
	portNumerical = intstr.FromInt(8125)
)

func TestFlattenNetworkPolicyIngressPorts(t *testing.T) {

	cases := []struct {
		Input          []networkingv1.NetworkPolicyPort
		ExpectedOutput []interface{}
	}{
		{
			[]networkingv1.NetworkPolicyPort{{
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
			[]networkingv1.NetworkPolicyPort{{
				Port: &portName,
			}},
			[]interface{}{
				map[string]interface{}{
					"port": "http",
				},
			},
		},
		{
			[]networkingv1.NetworkPolicyPort{{
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
			[]networkingv1.NetworkPolicyPort{{}},
			[]interface{}{map[string]interface{}{}},
		},
		{
			[]networkingv1.NetworkPolicyPort{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenNetworkPolicyV1Ports(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandNetworkPolicyIngressPorts(t *testing.T) {

	cases := []struct {
		Input          []interface{}
		ExpectedOutput *[]networkingv1.NetworkPolicyPort
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"port":     "http",
					"protocol": "TCP",
				},
			},
			&[]networkingv1.NetworkPolicyPort{{
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
			&[]networkingv1.NetworkPolicyPort{{
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
			&[]networkingv1.NetworkPolicyPort{{
				Port:     &portNumerical,
				Protocol: &protoUDP,
			}},
		},
		{
			[]interface{}{map[string]interface{}{}},
			&[]networkingv1.NetworkPolicyPort{{}},
		},
		{
			[]interface{}{},
			&[]networkingv1.NetworkPolicyPort{},
		},
	}

	for _, tc := range cases {
		output := expandNetworkPolicyV1Ports(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
