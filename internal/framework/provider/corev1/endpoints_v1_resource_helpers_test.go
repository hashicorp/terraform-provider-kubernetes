// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPreserveExplicitEmptyStrings(t *testing.T) {
	address := func(ip string, hostname, nodeName types.String) EndpointsAddressModel {
		return EndpointsAddressModel{IP: types.StringValue(ip), Hostname: hostname, NodeName: nodeName}
	}
	port := func(number int64, protocol string, name types.String) EndpointsPortModel {
		return EndpointsPortModel{Port: types.Int64Value(number), Protocol: types.StringValue(protocol), Name: name}
	}
	empty, null, host := types.StringValue(""), types.StringNull(), types.StringValue("host")

	cases := map[string]struct {
		prior, current, want []EndpointsSubsetModel
	}{
		"same subset": {
			prior: []EndpointsSubsetModel{{
				Address:         []EndpointsAddressModel{address("10.0.0.1", empty, empty), address("10.0.0.2", null, null)},
				NotReadyAddress: []EndpointsAddressModel{address("10.0.0.3", empty, null)},
				Port:            []EndpointsPortModel{port(80, "TCP", empty), port(81, "TCP", null)},
			}},
			current: []EndpointsSubsetModel{{
				Address: []EndpointsAddressModel{
					address("10.0.0.1", null, null),
					address("10.0.0.2", null, null),
					// Same IP as a prior not-ready address: readiness is part of the key.
					address("10.0.0.3", null, null),
					// A value now set by the API is never replaced.
					address("10.0.0.9", host, null),
				},
				NotReadyAddress: []EndpointsAddressModel{address("10.0.0.3", null, null)},
				Port:            []EndpointsPortModel{port(81, "TCP", null), port(80, "TCP", null)},
			}},
			want: []EndpointsSubsetModel{{
				Address: []EndpointsAddressModel{
					address("10.0.0.1", empty, empty),
					address("10.0.0.2", null, null),
					address("10.0.0.3", null, null),
					address("10.0.0.9", host, null),
				},
				NotReadyAddress: []EndpointsAddressModel{address("10.0.0.3", empty, null)},
				Port:            []EndpointsPortModel{port(81, "TCP", null), port(80, "TCP", empty)},
			}},
		},
		"same IP and port number in two subsets": {
			prior: []EndpointsSubsetModel{
				{Address: []EndpointsAddressModel{address("10.0.0.1", empty, empty)}, Port: []EndpointsPortModel{port(80, "TCP", empty)}},
				{Address: []EndpointsAddressModel{address("10.0.0.1", null, null)}, Port: []EndpointsPortModel{port(80, "UDP", null)}},
			},
			current: []EndpointsSubsetModel{
				{Address: []EndpointsAddressModel{address("10.0.0.1", null, null)}, Port: []EndpointsPortModel{port(80, "UDP", null)}},
				{Address: []EndpointsAddressModel{address("10.0.0.1", null, null)}, Port: []EndpointsPortModel{port(80, "TCP", null)}},
			},
			want: []EndpointsSubsetModel{
				{Address: []EndpointsAddressModel{address("10.0.0.1", null, null)}, Port: []EndpointsPortModel{port(80, "UDP", null)}},
				{Address: []EndpointsAddressModel{address("10.0.0.1", empty, empty)}, Port: []EndpointsPortModel{port(80, "TCP", empty)}},
			},
		},
		"subset ports changed outside Terraform": {
			prior: []EndpointsSubsetModel{
				{Address: []EndpointsAddressModel{address("10.0.0.1", empty, empty)}, Port: []EndpointsPortModel{port(80, "TCP", empty)}},
			},
			current: []EndpointsSubsetModel{
				{Address: []EndpointsAddressModel{address("10.0.0.1", null, null)}, Port: []EndpointsPortModel{port(80, "TCP", null), port(81, "TCP", null)}},
			},
			want: []EndpointsSubsetModel{
				{Address: []EndpointsAddressModel{address("10.0.0.1", null, null)}, Port: []EndpointsPortModel{port(80, "TCP", null), port(81, "TCP", null)}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			preserveExplicitEmptyStrings(tc.prior, tc.current)
			if !reflect.DeepEqual(tc.current, tc.want) {
				t.Errorf("got  %#v\nwant %#v", tc.current, tc.want)
			}
		})
	}
}

func TestNonEmptyEndpointsSubsets(t *testing.T) {
	addressType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"hostname": types.StringType, "ip": types.StringType, "node_name": types.StringType,
	}}
	portType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType, "port": types.Int64Type, "protocol": types.StringType,
	}}
	subsetTypes := map[string]attr.Type{
		"address":           types.SetType{ElemType: addressType},
		"not_ready_address": types.SetType{ElemType: addressType},
		"port":              types.SetType{ElemType: portType},
	}
	address := types.ObjectValueMust(addressType.AttrTypes, map[string]attr.Value{
		"hostname": types.StringNull(), "ip": types.StringValue("10.0.0.4"), "node_name": types.StringNull(),
	})
	port := types.ObjectValueMust(portType.AttrTypes, map[string]attr.Value{
		"name": types.StringNull(), "port": types.Int64Value(80), "protocol": types.StringValue("TCP"),
	})
	emptyAddresses := types.SetValueMust(addressType, []attr.Value{})
	someAddresses := types.SetValueMust(addressType, []attr.Value{address})
	subset := func(addresses, notReady, ports attr.Value) types.Object {
		return types.ObjectValueMust(subsetTypes, map[string]attr.Value{
			"address": addresses, "not_ready_address": notReady, "port": ports,
		})
	}
	noPorts := types.SetValueMust(portType, []attr.Value{})
	subsetType := types.ObjectType{AttrTypes: subsetTypes}
	somePorts := types.SetValueMust(portType, []attr.Value{port})

	for name, tc := range map[string]struct {
		value types.Object
		kept  bool
	}{
		"empty subset":            {value: subset(emptyAddresses, emptyAddresses, noPorts)},
		"null nested blocks":      {value: subset(types.SetNull(addressType), types.SetNull(addressType), types.SetNull(portType))},
		"ports only":              {value: subset(emptyAddresses, emptyAddresses, somePorts), kept: true},
		"address":                 {value: subset(someAddresses, emptyAddresses, somePorts), kept: true},
		"not_ready_address only":  {value: subset(emptyAddresses, someAddresses, noPorts), kept: true},
		"unknown dynamic address": {value: subset(types.SetUnknown(addressType), emptyAddresses, noPorts), kept: true},
	} {
		t.Run(name, func(t *testing.T) {
			got := nonEmptyEndpointsSubsets(context.Background(), types.SetValueMust(subsetType, []attr.Value{tc.value}))
			if kept := len(got.Elements()) == 1; kept != tc.kept {
				t.Errorf("kept = %t, want %t", kept, tc.kept)
			}
		})
	}
}

func TestExpandEndpointsSubsetsDropsEmptySubset(t *testing.T) {
	address := []EndpointsAddressModel{{IP: types.StringValue("10.0.0.4"), Hostname: types.StringNull(), NodeName: types.StringNull()}}
	ports := []EndpointsPortModel{{Port: types.Int64Value(80), Name: types.StringNull(), Protocol: types.StringValue("TCP")}}

	for name, tc := range map[string]struct {
		in   []EndpointsSubsetModel
		want int
	}{
		"only empty subset":          {in: []EndpointsSubsetModel{{}}, want: 0},
		"empty and populated subset": {in: []EndpointsSubsetModel{{}, {Address: address, Port: ports}}, want: 1},
		"ports only is sent":         {in: []EndpointsSubsetModel{{Port: ports}}, want: 1},
	} {
		t.Run(name, func(t *testing.T) {
			got := expandEndpointsSubsets(tc.in)
			if got == nil || len(got) != tc.want {
				t.Errorf("got %#v, want %d non-nil subsets", got, tc.want)
			}
		})
	}
}
