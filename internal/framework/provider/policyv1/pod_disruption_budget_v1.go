// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	policyclient "k8s.io/client-go/kubernetes/typed/policy/v1"
)

var (
	_ resource.Resource                = (*PodDisruptionBudgetV1)(nil)
	_ resource.ResourceWithConfigure   = (*PodDisruptionBudgetV1)(nil)
	_ resource.ResourceWithImportState = (*PodDisruptionBudgetV1)(nil)
)

type PodDisruptionBudgetV1 struct {
	SDKv2Meta func() any
}

func NewPodDisruptionBudgetV1() resource.Resource {
	return &PodDisruptionBudgetV1{}
}

func (r *PodDisruptionBudgetV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pod_disruption_budget_v1"
}

func (r *PodDisruptionBudgetV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok || meta == nil {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected a non-nil func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *PodDisruptionBudgetV1) client() (policyclient.PolicyV1Interface, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable.")
		return nil, nil, diags
	}
	// Resolve lazily: the mux configures the SDKv2 server independently.
	meta := r.SDKv2Meta()
	if meta == nil || (reflect.ValueOf(meta).Kind() == reflect.Pointer && reflect.ValueOf(meta).IsNil()) {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable.")
		return nil, nil, diags
	}
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
	conn, err := clients.MainClientset()
	if err != nil {
		diags.AddError("Kubernetes client error", err.Error())
		return nil, nil, diags
	}
	if conn == nil || conn.PolicyV1() == nil {
		diags.AddError("Kubernetes client error", "The Kubernetes policy/v1 client is unavailable.")
		return nil, nil, diags
	}
	return conn.PolicyV1(), filters, diags
}

func (r *PodDisruptionBudgetV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, _, err := podDisruptionBudgetIDParts(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func podDisruptionBudgetIDParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" ||
		strings.TrimSpace(parts[0]) != parts[0] || strings.TrimSpace(parts[1]) != parts[1] {
		return "", "", fmt.Errorf("Expected ID in the format namespace/name with non-empty namespace and name, got %q.", id)
	}
	return parts[0], parts[1], nil
}
