// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ resource.Resource                    = (*ResourceQuotaV1)(nil)
	_ resource.ResourceWithConfigure       = (*ResourceQuotaV1)(nil)
	_ resource.ResourceWithIdentity        = (*ResourceQuotaV1)(nil)
	_ resource.ResourceWithImportState     = (*ResourceQuotaV1)(nil)
	_ resource.ResourceWithMoveState       = (*ResourceQuotaV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ResourceQuotaV1)(nil)
)

type ResourceQuotaV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in ProviderData by
	// internal/framework/provider/provider_configure.go. Asserting to a func returning
	// kubernetes.KubeClientsets compiles but panics at runtime, so assert the result in meta().
	SDKv2Meta func() any
}

func NewResourceQuotaV1() resource.Resource {
	return &ResourceQuotaV1{}
}

// Metadata implements [resource.Resource].
func (r *ResourceQuotaV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_quota_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (r *ResourceQuotaV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// meta resolves the SDKv2 provider metadata into the two interfaces this resource needs. The
// call is deferred until the request because the mux configures the SDKv2 provider
// independently and its meta is not populated until then. Configure returns early when
// ProviderData is nil, so SDKv2Meta can legitimately be nil here.
func (r *ResourceQuotaV1) meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
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

const (
	resourceQuotaAPIVersion = "v1"
	resourceQuotaKind       = "ResourceQuota"

	deprecatedResourceQuotaTypeName      = "kubernetes_resource_quota"
	deprecatedResourceQuotaSchemaVersion = 0

	resourceQuotaMoveErrSummary = "Unable to move kubernetes_resource_quota state"
)

// ImportState implements [resource.ResourceWithImportState].
//
// SDKv2 imported by passing the "namespace/name" id straight through and had no identity.
// Both forms are accepted here: the id, and the identity added by this migration. Read
// validates the id and fills in state and identity, so this only has to set "id".
//
// An identity without a namespace is rejected rather than defaulting to "default"
// (K8S-MIGRATE-022): a silent default would import a different object than the one the user
// named. The namespace attribute is OptionalForImport only to match the shared identity schema.
func (r *ResourceQuotaV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" {
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.Kind.ValueString() != resourceQuotaKind || identity.APIVersion.ValueString() != resourceQuotaAPIVersion {
			resp.Diagnostics.AddError("Invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					resourceQuotaKind, resourceQuotaAPIVersion, identity.Kind.ValueString(), identity.APIVersion.ValueString()))
			return
		}
		if identity.Namespace.ValueString() == "" || identity.Name.ValueString() == "" {
			resp.Diagnostics.AddError("Invalid identity import",
				"Both namespace and name are required to import a resource quota by identity.")
			return
		}
		id = kubernetes.BuildId(metav1.ObjectMeta{Namespace: identity.Namespace.ValueString(), Name: identity.Name.ValueString()})
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// SDKv2 resource_quota_v1 declared no identity, so state written by every released provider
// carries identity_schema_version 0 and no identity. Terraform still asks for an upgrade
// whenever the stored version differs from the declared one, and the framework rejects an
// empty answer. common.UpgradeNamespacedIdentity returns an all-null identity for that case,
// which Read then fills in.
func (r *ResourceQuotaV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(resourceQuotaKind, resourceQuotaAPIVersion)
}

// sdkv2ResourceQuotaStateV0 is kubernetes_resource_quota state as SDKv2 stores it at schema
// version 0. It is frozen: it describes stored JSON, not this resource's schema.
//
// Types follow what SDKv2 3.4.0 wrote, checked against a live apply. Strings are "" when unset.
// Collections keep their null-vs-empty distinction: an unset scopes is null with a configured
// spec but [] when the whole spec is omitted, so they stay Go slices and maps, where nil and
// empty differ.
type sdkv2ResourceQuotaStateV0 struct {
	ID       string                             `json:"id"`
	Metadata []common.SDKv2NamespacedMetadataV0 `json:"metadata"`
	Spec     []sdkv2ResourceQuotaSpecV0         `json:"spec"`
	Timeouts *sdkv2ResourceQuotaTimeoutsV0      `json:"timeouts"`
}

type sdkv2ResourceQuotaSpecV0 struct {
	Hard          map[string]string                   `json:"hard"`
	Scopes        []string                            `json:"scopes"`
	ScopeSelector []sdkv2ResourceQuotaScopeSelectorV0 `json:"scope_selector"`
}

type sdkv2ResourceQuotaScopeSelectorV0 struct {
	MatchExpression []sdkv2ResourceQuotaMatchExpressionV0 `json:"match_expression"`
}

type sdkv2ResourceQuotaMatchExpressionV0 struct {
	ScopeName string   `json:"scope_name"`
	Operator  string   `json:"operator"`
	Values    []string `json:"values"`
}

// Create and Update are "" when the stored value is null, which is what an empty
// `timeouts {}` block produces. Both mean unset.
type sdkv2ResourceQuotaTimeoutsV0 struct {
	Create string `json:"create"`
	Update string `json:"update"`
}

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_resource_quota.example
//	  to   = kubernetes_resource_quota_v1.example
//	}
//
// the supported route off the deprecated alias, which stays on SDKv2. The source is decoded
// from raw JSON into a frozen struct rather than through a SourceSchema, so it does not
// follow this resource's schema as that changes.
func (r *ResourceQuotaV1) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	timeoutsAttrTypes := timeoutsAttributeTypes(schemaResp.Schema)

	return []resource.StateMover{
		{
			// No SourceSchema: the framework would decode against it before calling us.
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Do not guard on req.SourceIdentitySchemaVersion. terraform-plugin-go never
				// copies it off the wire, so it is always 0.
				if req.SourceTypeName != deprecatedResourceQuotaTypeName ||
					req.SourceSchemaVersion != deprecatedResourceQuotaSchemaVersion ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					// Skipping returns no state and no diagnostics, and the framework then
					// fails the move without saying why. Leave a reason in the log.
					tflog.Debug(ctx, "MoveState: not a kubernetes_resource_quota v0 source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
					resp.Diagnostics.AddError(resourceQuotaMoveErrSummary,
						"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
					return
				}

				var prior sdkv2ResourceQuotaStateV0
				if err := json.Unmarshal(req.SourceRawState.JSON, &prior); err != nil {
					resp.Diagnostics.AddError(resourceQuotaMoveErrSummary, fmt.Sprintf("Could not decode the source state: %s", err))
					return
				}
				if prior.ID == "" {
					resp.Diagnostics.AddError(resourceQuotaMoveErrSummary, "The source state has an empty id.")
					return
				}
				if len(prior.Metadata) != 1 {
					resp.Diagnostics.AddError(resourceQuotaMoveErrSummary,
						fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(prior.Metadata)))
					return
				}
				if len(prior.Spec) > 1 {
					resp.Diagnostics.AddError(resourceQuotaMoveErrSummary,
						fmt.Sprintf("Expected at most 1 spec element in the source state, got %d.", len(prior.Spec)))
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

				// SDKv2 writes "" for an unset generate_name, where the framework needs null.
				// Left as "", a plan with -refresh=false compares "" against null and
				// RequiresReplace recreates the quota.
				meta.GenerateName = types.StringNull()
				if m.GenerateName != "" {
					meta.GenerateName = types.StringValue(m.GenerateName)
				}

				spec, specDiags := sdkv2ResourceQuotaSpecToFramework(ctx, prior.Spec)
				resp.Diagnostics.Append(specDiags...)

				timeoutsValue, tDiags := sdkv2ResourceQuotaTimeoutsToFramework(timeoutsAttrTypes, prior.Timeouts)
				resp.Diagnostics.Append(tDiags...)
				if resp.Diagnostics.HasError() {
					return
				}

				state := ResourceQuotaV1Model{
					ID:       types.StringValue(prior.ID),
					Metadata: []common.NamespacedMetadataModel{meta},
					Spec:     spec,
					Timeouts: timeoutsValue,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				// Build the identity rather than decoding req.SourceIdentity: api_version and
				// kind are constants and the name and namespace are in the state just moved.
				// This also covers state written before identity existed.
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						APIVersion: types.StringValue(resourceQuotaAPIVersion),
						Kind:       types.StringValue(resourceQuotaKind),
						Name:       types.StringValue(m.Name),
					},
					Namespace: types.StringValue(m.Namespace),
				})...)
			},
		},
	}
}

// sdkv2ResourceQuotaSpecToFramework converts the stored spec without guessing user intent: a
// nil collection stays null and an empty one stays empty, because the move cannot see the
// configuration that produced them.
func sdkv2ResourceQuotaSpecToFramework(ctx context.Context, in []sdkv2ResourceQuotaSpecV0) ([]ResourceQuotaSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(in) == 0 {
		return []ResourceQuotaSpecModel{}, diags
	}
	s := in[0]

	out := ResourceQuotaSpecModel{
		Hard:   types.MapNull(common.QuantityType{}),
		Scopes: types.SetNull(types.StringType),
	}

	if s.Hard != nil {
		elems := make(map[string]attr.Value, len(s.Hard))
		for k, v := range s.Hard {
			elems[k] = common.NewQuantityValue(v)
		}
		hard, d := types.MapValue(common.QuantityType{}, elems)
		diags.Append(d...)
		out.Hard = hard
	}

	if s.Scopes != nil {
		scopes, d := types.SetValueFrom(ctx, types.StringType, s.Scopes)
		diags.Append(d...)
		out.Scopes = scopes
	}

	// SDKv2 stores an absent scope_selector as [], which is also what the framework holds
	// for an omitted block, so the length carries over unchanged.
	out.ScopeSelector = make([]ResourceQuotaScopeSelectorModel, len(s.ScopeSelector))
	for i, sel := range s.ScopeSelector {
		exprs := make([]ResourceQuotaMatchExpressionModel, len(sel.MatchExpression))
		for j, e := range sel.MatchExpression {
			values := types.SetNull(types.StringType)
			if e.Values != nil {
				v, d := types.SetValueFrom(ctx, types.StringType, e.Values)
				diags.Append(d...)
				values = v
			}
			exprs[j] = ResourceQuotaMatchExpressionModel{
				ScopeName: types.StringValue(e.ScopeName),
				Operator:  types.StringValue(e.Operator),
				Values:    values,
			}
		}
		out.ScopeSelector[i] = ResourceQuotaScopeSelectorModel{MatchExpression: exprs}
	}

	return []ResourceQuotaSpecModel{out}, diags
}

func sdkv2ResourceQuotaTimeoutsToFramework(attrTypes map[string]attr.Type, t *sdkv2ResourceQuotaTimeoutsV0) (timeouts.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	if attrTypes == nil {
		diags.AddError(resourceQuotaMoveErrSummary, "The timeouts block has no attribute types. This is a bug in the provider.")
		return timeouts.Value{}, diags
	}
	if t == nil {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}, nil
	}

	attrs := make(map[string]attr.Value, len(attrTypes))
	for k := range attrTypes {
		attrs[k] = types.StringNull()
	}
	if t.Create != "" {
		attrs["create"] = types.StringValue(t.Create)
	}
	if t.Update != "" {
		attrs["update"] = types.StringValue(t.Update)
	}
	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return timeouts.Value{Object: obj}, diags
}
