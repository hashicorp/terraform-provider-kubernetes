// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type EndpointsV1Model struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Subset   []EndpointsSubsetModel           `tfsdk:"subset"`
}

type EndpointsSubsetModel struct {
	Address         []EndpointsAddressModel `tfsdk:"address"`
	NotReadyAddress []EndpointsAddressModel `tfsdk:"not_ready_address"`
	Port            []EndpointsPortModel    `tfsdk:"port"`
}

type EndpointsAddressModel struct {
	Hostname types.String `tfsdk:"hostname"`
	IP       types.String `tfsdk:"ip"`
	NodeName types.String `tfsdk:"node_name"`
}

type EndpointsPortModel struct {
	Name     types.String `tfsdk:"name"`
	Port     types.Int64  `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
}

type sdkv2EndpointsStateV0 struct {
	ID       string                             `json:"id"`
	Metadata []common.SDKv2NamespacedMetadataV0 `json:"metadata"`
	Subset   []sdkv2EndpointsSubsetV0           `json:"subset"`
}

type sdkv2EndpointsSubsetV0 struct {
	Address         []sdkv2EndpointsAddressV0 `json:"address"`
	NotReadyAddress []sdkv2EndpointsAddressV0 `json:"not_ready_address"`
	Port            []sdkv2EndpointsPortV0    `json:"port"`
}

type sdkv2EndpointsAddressV0 struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	NodeName string `json:"node_name"`
}

type sdkv2EndpointsPortV0 struct {
	Name     string `json:"name"`
	Port     int64  `json:"port"`
	Protocol string `json:"protocol"`
}
