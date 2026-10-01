// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*PodV1)(nil)
	_ resource.ResourceWithConfigure       = (*PodV1)(nil)
	_ resource.ResourceWithIdentity        = (*PodV1)(nil)
	_ resource.ResourceWithImportState     = (*PodV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*PodV1)(nil)
	_ resource.ResourceWithMoveState       = (*PodV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*PodV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*PodV1)(nil)
)

const (
	podAPIVersion = "v1"
	podKind       = "Pod"
)

type PodV1 struct {
	SDKv2Meta func() any
}

func NewPodV1() resource.Resource {
	return &PodV1{}
}

func (p *PodV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pod_v1"
}

func (p *PodV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	p.SDKv2Meta = meta
}

func (p *PodV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if p.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "The Kubernetes provider metadata is unavailable.")
		return nil, nil, diags
	}
	meta := p.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T. This is a bug in the provider.", meta))
		return nil, nil, diags
	}
	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T. This is a bug in the provider.", meta))
		return nil, nil, diags
	}
	return clients, filters, diags
}
