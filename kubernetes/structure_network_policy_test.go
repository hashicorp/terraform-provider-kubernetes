package kubernetes

import (
	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/networking/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"testing"
)

var (
	protoTcp      = api.ProtocolTCP
	protoUdp      = api.ProtocolUDP
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
				Protocol: &protoTcp,
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
				Protocol: &protoUdp,
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
		output := flattenNetworkPolicyIngressPorts(tc.Input)
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
				Protocol: &protoTcp,
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
				Protocol: &protoUdp,
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
		output := expandNetworkPolicyIngressPorts(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
