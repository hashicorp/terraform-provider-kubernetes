// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*JobV1)(nil)
	_ resource.ResourceWithConfigure       = (*JobV1)(nil)
	_ resource.ResourceWithIdentity        = (*JobV1)(nil)
	_ resource.ResourceWithImportState     = (*JobV1)(nil)
	_ resource.ResourceWithMoveState       = (*JobV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*JobV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*JobV1)(nil)
)

type JobV1 struct {
	SDKv2Meta func() any
}

func NewJobV1() resource.Resource {
	return &JobV1{}
}

func (r *JobV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_v1"
}

func (r *JobV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T.", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *JobV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "The Kubernetes provider metadata is unavailable.")
		return nil, nil, diags
	}
	meta := r.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T.", meta))
		return nil, nil, diags
	}
	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T.", meta))
		return nil, nil, diags
	}
	return clients, filters, diags
}
