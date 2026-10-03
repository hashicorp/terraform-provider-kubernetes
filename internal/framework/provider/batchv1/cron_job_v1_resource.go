// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	cronJobAPIVersion = "batch/v1"
	cronJobKind       = "CronJob"
)

var (
	_ resource.Resource                    = (*CronJobV1)(nil)
	_ resource.ResourceWithConfigure       = (*CronJobV1)(nil)
	_ resource.ResourceWithIdentity        = (*CronJobV1)(nil)
	_ resource.ResourceWithImportState     = (*CronJobV1)(nil)
	_ resource.ResourceWithMoveState       = (*CronJobV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*CronJobV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*CronJobV1)(nil)
)

type CronJobV1 struct {
	SDKv2Meta func() any
}

func NewCronJobV1() resource.Resource {
	return &CronJobV1{}
}

func (r *CronJobV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cron_job_v1"
}

func (r *CronJobV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CronJobV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "Kubernetes provider metadata is unavailable.")
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

func (r *CronJobV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" && req.Identity != nil && !req.Identity.Raw.IsNull() {
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.APIVersion.ValueString() != cronJobAPIVersion || identity.Kind.ValueString() != cronJobKind {
			resp.Diagnostics.AddError("Invalid CronJob identity", "Expected api_version batch/v1 and kind CronJob.")
			return
		}
		namespace := identity.Namespace.ValueString()
		if namespace == "" {
			namespace = "default"
		}
		id = namespace + "/" + identity.Name.ValueString()
	}
	if _, _, err := cronJobIDParts(id); err != nil {
		resp.Diagnostics.AddError("Invalid CronJob import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func (r *CronJobV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(cronJobKind, cronJobAPIVersion)
}

func cronJobIDParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected namespace/name, got %q", id)
	}
	return parts[0], parts[1], nil
}

func cronJobIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(cronJobAPIVersion),
			Kind:       types.StringValue(cronJobKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}
