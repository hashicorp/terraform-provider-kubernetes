// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type IngressV1Model struct {
	ID                  types.String                     `tfsdk:"id"`
	Metadata            []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec                []IngressV1SpecModel             `tfsdk:"spec"`
	Status              types.List                       `tfsdk:"status"`
	WaitForLoadBalancer types.Bool                       `tfsdk:"wait_for_load_balancer"`
	Timeouts            timeouts.Value                   `tfsdk:"timeouts"`
}

type IngressV1SpecModel struct {
	IngressClassName types.String           `tfsdk:"ingress_class_name"`
	DefaultBackend   *IngressV1BackendModel `tfsdk:"default_backend"`
	Rule             []IngressV1RuleModel   `tfsdk:"rule"`
	TLS              []IngressV1TLSModel    `tfsdk:"tls"`
}

type IngressV1BackendModel struct {
	Resource *IngressV1ResourceBackendModel `tfsdk:"resource"`
	Service  *IngressV1ServiceBackendModel  `tfsdk:"service"`
}

type IngressV1ResourceBackendModel struct {
	APIGroup types.String `tfsdk:"api_group"`
	Kind     types.String `tfsdk:"kind"`
	Name     types.String `tfsdk:"name"`
}

type IngressV1ServiceBackendModel struct {
	Name types.String        `tfsdk:"name"`
	Port *IngressV1PortModel `tfsdk:"port"`
}

type IngressV1PortModel struct {
	Name   types.String `tfsdk:"name"`
	Number types.Int64  `tfsdk:"number"`
}

type IngressV1RuleModel struct {
	Host types.String        `tfsdk:"host"`
	HTTP *IngressV1HTTPModel `tfsdk:"http"`
}

type IngressV1HTTPModel struct {
	Path []IngressV1PathModel `tfsdk:"path"`
}

type IngressV1PathModel struct {
	Path     types.String           `tfsdk:"path"`
	PathType types.String           `tfsdk:"path_type"`
	Backend  *IngressV1BackendModel `tfsdk:"backend"`
}

type IngressV1TLSModel struct {
	Hosts      types.List   `tfsdk:"hosts"`
	SecretName types.String `tfsdk:"secret_name"`
}
