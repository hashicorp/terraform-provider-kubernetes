// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*LimitRangeV1)(nil)
	_ resource.ResourceWithConfigure       = (*LimitRangeV1)(nil)
	_ resource.ResourceWithIdentity        = (*LimitRangeV1)(nil)
	_ resource.ResourceWithImportState     = (*LimitRangeV1)(nil)
	_ resource.ResourceWithMoveState       = (*LimitRangeV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*LimitRangeV1)(nil)
)

// LimitRangeV1 is the Framework implementation of kubernetes_limit_range_v1.
type LimitRangeV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in
	// ProviderData by internal/framework/provider/provider_configure.go.
	// Go function types are invariant — assert the *result*, not the function type.
	SDKv2Meta func() any
}

func NewLimitRangeV1() resource.Resource {
	return &LimitRangeV1{}
}

// Metadata implements [resource.Resource].
func (r *LimitRangeV1) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_limit_range_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (r *LimitRangeV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	r.SDKv2Meta = sdkv2Meta
}

// meta resolves the SDKv2 provider metadata into the two interfaces this resource needs.
// Called per-request because the mux configures the SDKv2 provider independently.
func (r *LimitRangeV1) meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics

	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured",
			"The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, nil, diags
	}

	meta := r.SDKv2Meta()
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

// ImportState implements [resource.ResourceWithImportState].
//
// The import ID is "namespace/name", matching SDKv2's ImportStatePassthroughContext.
func (r *LimitRangeV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// sdkv2LimitRangeStateV0 is the frozen shape of kubernetes_limit_range (the deprecated
// alias) state as SDKv2 writes it at schema version 0. Used only by MoveState.
type sdkv2LimitRangeStateV0 struct {
	ID       string                         `json:"id"`
	Metadata []sdkv2LimitRangeMetadataV0    `json:"metadata"`
	Spec     []sdkv2LimitRangeSpecV0        `json:"spec"`
}

type sdkv2LimitRangeMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type sdkv2LimitRangeSpecV0 struct {
	Limit []sdkv2LimitRangeItemV0 `json:"limit"`
}

type sdkv2LimitRangeItemV0 struct {
	Default              map[string]string `json:"default"`
	DefaultRequest       map[string]string `json:"default_request"`
	Max                  map[string]string `json:"max"`
	MaxLimitRequestRatio map[string]string `json:"max_limit_request_ratio"`
	Min                  map[string]string `json:"min"`
	Type                 string            `json:"type"`
}

const limitRangeMoveStateErrSummary = "Unable to move kubernetes_limit_range state"

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_limit_range.example
//	  to   = kubernetes_limit_range_v1.example
//	}
//
// the supported migration path off the deprecated alias, which stays on SDKv2.
// Terraform allows a cross-type move only when the destination implements this.
//
// The source is decoded from raw JSON into a frozen struct rather than through a
// SourceSchema so it does not follow this resource's schema as that changes.
func (r *LimitRangeV1) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			// No SourceSchema: the framework decodes against it *before* calling us,
			// which fails when raw identity is absent — the normal case.
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Do not guard on req.SourceIdentitySchemaVersion: terraform-plugin-go
				// never copies it off the wire (v0.31.0), so it is always 0.
				if req.SourceTypeName != "kubernetes_limit_range" ||
					req.SourceSchemaVersion != 0 ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					tflog.Debug(ctx, "MoveState: not a kubernetes_limit_range v0 source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
					resp.Diagnostics.AddError(limitRangeMoveStateErrSummary,
						"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
					return
				}

				var prior sdkv2LimitRangeStateV0
				if err := json.Unmarshal(req.SourceRawState.JSON, &prior); err != nil {
					resp.Diagnostics.AddError(limitRangeMoveStateErrSummary,
						fmt.Sprintf("Could not decode the source state: %s", err))
					return
				}
				if prior.ID == "" {
					resp.Diagnostics.AddError(limitRangeMoveStateErrSummary,
						"The source state has an empty id.")
					return
				}
				if len(prior.Metadata) != 1 {
					resp.Diagnostics.AddError(limitRangeMoveStateErrSummary,
						fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(prior.Metadata)))
					return
				}
				m := prior.Metadata[0]

				var diags diag.Diagnostics

				meta := common.NamespacedMetadataModel{
					MetadataModel: common.MetadataModel{
						MetadataBase: common.MetadataBase{
							Name:            types.StringValue(m.Name),
							Generation:      types.Int64Value(m.Generation),
							ResourceVersion: types.StringValue(m.ResourceVersion),
							UID:             types.StringValue(m.UID),
						},
					},
					Namespace: types.StringValue(m.Namespace),
				}
				meta.Labels, diags = sdkv2MapToFramework(ctx, m.Labels)
				resp.Diagnostics.Append(diags...)
				meta.Annotations, diags = sdkv2MapToFramework(ctx, m.Annotations)
				resp.Diagnostics.Append(diags...)

				// SDKv2 writes "" for an unset generate_name; the framework uses null.
				// Left as "", plan -refresh=false would compare "" vs null and
				// RequiresReplace would recreate the resource.
				meta.GenerateName = types.StringNull()
				if m.GenerateName != "" {
					meta.GenerateName = types.StringValue(m.GenerateName)
				}

				// Convert spec items, preserving the null-vs-empty distinction for each map.
				spec, specDiags := sdkv2LimitRangeSpecToFramework(ctx, prior.Spec)
				resp.Diagnostics.Append(specDiags...)
				if resp.Diagnostics.HasError() {
					return
				}

				state := LimitRangeV1Model{
					ID:       types.StringValue(prior.ID),
					Metadata: []common.NamespacedMetadataModel{meta},
					Spec:     spec,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				// Build identity from the state just moved rather than decoding
				// req.SourceIdentity — api_version and kind are constants, and a
				// source written before identity existed carries no identity at all.
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						APIVersion: types.StringValue(limitRangeAPIVersion),
						Kind:       types.StringValue(limitRangeKind),
						Name:       types.StringValue(m.Name),
					},
					Namespace: types.StringValue(m.Namespace),
				})...)
			},
		},
	}
}

// sdkv2LimitRangeSpecToFramework converts the frozen SDKv2 spec shape into Framework models.
func sdkv2LimitRangeSpecToFramework(ctx context.Context, sdkSpec []sdkv2LimitRangeSpecV0) ([]LimitRangeSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(sdkSpec) == 0 {
		return nil, diags
	}
	s := sdkSpec[0]
	items := make([]LimitRangeItemModel, len(s.Limit))
	for i, l := range s.Limit {
		item := LimitRangeItemModel{
			Type: types.StringValue(l.Type),
		}
		var d diag.Diagnostics

		item.Default, d = sdkv2MapToFramework(ctx, l.Default)
		diags.Append(d...)
		item.DefaultRequest, d = sdkv2MapToFramework(ctx, l.DefaultRequest)
		diags.Append(d...)
		item.Max, d = sdkv2MapToFramework(ctx, l.Max)
		diags.Append(d...)
		item.MaxLimitRequestRatio, d = sdkv2MapToFramework(ctx, l.MaxLimitRequestRatio)
		diags.Append(d...)
		item.Min, d = sdkv2MapToFramework(ctx, l.Min)
		diags.Append(d...)

		items[i] = item
	}
	return []LimitRangeSpecModel{{Limit: items}}, diags
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// The SDKv2 resource never declared an Identity field, so all existing state carries
// identity_schema_version 0 with no identity at all. common.UpgradeNamespacedIdentity
// handles both cases: present (decode name+namespace) and absent (return all-null so
// Read can populate it later). Framework v1.16.1+ is required for the all-null path.
func (r *LimitRangeV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(limitRangeKind, limitRangeAPIVersion)
}
