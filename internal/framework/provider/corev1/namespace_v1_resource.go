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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*NamespaceV1)(nil)
	_ resource.ResourceWithConfigure       = (*NamespaceV1)(nil)
	_ resource.ResourceWithIdentity        = (*NamespaceV1)(nil)
	_ resource.ResourceWithImportState     = (*NamespaceV1)(nil)
	_ resource.ResourceWithMoveState       = (*NamespaceV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*NamespaceV1)(nil)
)

type NamespaceV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in
	// ProviderData by internal/framework/provider/provider_configure.go. Go
	// function types are invariant, so asserting to func() kubernetes.KubeClientsets
	// compiles but panics at runtime. Assert the *result* instead — see meta().
	SDKv2Meta func() any
}

func NewNamespaceV1() resource.Resource {
	return &NamespaceV1{}
}

// Metadata implements [resource.Resource].
func (n *NamespaceV1) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (n *NamespaceV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	n.SDKv2Meta = sdkv2Meta
}

// sdkv2Meta resolves the SDKv2 provider metadata into the two interfaces this resource
// needs. The call is deferred until the request rather than made in Configure, because the
// mux server configures the SDKv2 provider independently and its meta is not populated
// until that happens.
//
// Everything is checked: Configure returns early when ProviderData is nil, so SDKv2Meta can
// legitimately be nil here, and an unexpected type must be a diagnostic rather than a panic
// that takes the plugin process down.
func (n *NamespaceV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics

	if n.SDKv2Meta == nil {
		diags.AddError("Provider not configured",
			"The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, nil, diags
	}

	meta := n.SDKv2Meta()
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
func (n *NamespaceV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

// sdkv2NamespaceStateV0 is kubernetes_namespace state as SDKv2 stores it at schema version
// 0. It is frozen: it describes stored JSON, not this resource's schema.
//
// Types follow what SDKv2 writes (checked against 2.37.1 and 3.2.1 state). Its legacy type
// system has no null for primitives, so an unset string is "" and an unset bool is false.
// Maps are the exception — unset is null, except after an import, which stores {} — so they
// stay Go maps, where nil and empty are distinct.
type sdkv2NamespaceStateV0 struct {
	ID                           string                     `json:"id"`
	WaitForDefaultServiceAccount bool                       `json:"wait_for_default_service_account"`
	Metadata                     []sdkv2NamespaceMetadataV0 `json:"metadata"`
	Timeouts                     *sdkv2NamespaceTimeoutsV0  `json:"timeouts"`
}

type sdkv2NamespaceMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

// Delete is "" when the stored value is null, which is what an empty `timeouts {}` block
// produces. Both mean unset.
type sdkv2NamespaceTimeoutsV0 struct {
	Delete string `json:"delete"`
}

// The source provider address is HOSTNAME/NAMESPACE/TYPE, and the hostname is ignored: it is
// the public registry for most users and a mirror or private registry for others. The leading
// slash anchors the match to a whole segment, so a fork such as nothashicorp/kubernetes does
// not match.
const sdkv2ProviderAddressSuffix = "/hashicorp/kubernetes"

const moveStateErrSummary = "Unable to move kubernetes_namespace state"

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_namespace.example
//	  to   = kubernetes_namespace_v1.example
//	}
//
// the supported route off the deprecated alias, which stays on SDKv2. Terraform allows a
// cross-type move only when the destination provider implements this.
//
// The source is decoded from raw JSON into a frozen struct rather than through a
// SourceSchema, so it does not follow this resource's schema as that changes. A future SDKv2
// schema version needs its own struct and a SourceSchemaVersion branch, not edits to this one.
func (n *NamespaceV1) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	n.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	timeoutsAttrTypes := timeoutsAttributeTypes(schemaResp.Schema)

	return []resource.StateMover{
		{
			// No SourceSchema: the framework would decode against it before calling us.
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Do not guard on req.SourceIdentitySchemaVersion. terraform-plugin-go never
				// copies it off the wire (still true in v0.31.0), so it is always 0.
				if req.SourceTypeName != "kubernetes_namespace" ||
					req.SourceSchemaVersion != 0 ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					// Skipping means returning no state and no diagnostics, and the framework
					// then fails the move without saying why. Leave a reason in the log.
					tflog.Debug(ctx, "MoveState: not a kubernetes_namespace v0 source, skipping", map[string]any{
						"source_type_name":        req.SourceTypeName,
						"source_schema_version":   req.SourceSchemaVersion,
						"source_provider_address": req.SourceProviderAddress,
					})
					return
				}

				if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
					resp.Diagnostics.AddError(moveStateErrSummary,
						"The source state has no JSON data. Flatmap state predates Terraform 0.12 and is not supported.")
					return
				}

				var prior sdkv2NamespaceStateV0
				if err := json.Unmarshal(req.SourceRawState.JSON, &prior); err != nil {
					resp.Diagnostics.AddError(moveStateErrSummary, fmt.Sprintf("Could not decode the source state: %s", err))
					return
				}
				if prior.ID == "" {
					resp.Diagnostics.AddError(moveStateErrSummary, "The source state has an empty id.")
					return
				}
				if len(prior.Metadata) != 1 {
					resp.Diagnostics.AddError(moveStateErrSummary,
						fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(prior.Metadata)))
					return
				}
				m := prior.Metadata[0]

				var diags diag.Diagnostics
				meta := common.MetadataModel{
					Name:            types.StringValue(m.Name),
					Generation:      types.Int64Value(m.Generation),
					ResourceVersion: types.StringValue(m.ResourceVersion),
					UID:             types.StringValue(m.UID),
				}
				meta.Labels, diags = sdkv2MapToFramework(ctx, m.Labels)
				resp.Diagnostics.Append(diags...)
				meta.Annotations, diags = sdkv2MapToFramework(ctx, m.Annotations)
				resp.Diagnostics.Append(diags...)

				// SDKv2 writes "" for an unset generate_name, where the framework needs null.
				// Left as "", a plan with -refresh=false compares "" against null and
				// RequiresReplace recreates the namespace.
				meta.GenerateName = types.StringNull()
				if m.GenerateName != "" {
					meta.GenerateName = types.StringValue(m.GenerateName)
				}

				timeoutsValue, diags := sdkv2TimeoutsToFramework(timeoutsAttrTypes, prior.Timeouts)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				state := NamespaceV1Model{
					ID:                           types.StringValue(prior.ID),
					WaitForDefaultServiceAccount: types.BoolValue(prior.WaitForDefaultServiceAccount),
					Metadata:                     []common.MetadataModel{meta},
					Timeouts:                     timeoutsValue,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				// Build the identity rather than decoding req.SourceIdentity: api_version and
				// kind are constants, and the name is in the state just moved. This also
				// covers state written before identity existed.
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, NamespaceResourceIdentity{
					APIVersion: types.StringValue(namespaceAPIVersion),
					Kind:       types.StringValue(namespaceKind),
					Name:       types.StringValue(m.Name),
				})...)
			},
		},
	}
}

// sdkv2MapToFramework keeps SDKv2's null-vs-empty distinction: a nil Go map (JSON null)
// becomes a typed null, {} stays a known empty map. Never leave the field unset instead — a
// zero-value types.Map carries no element type and fails to write to state.
func sdkv2MapToFramework(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

// timeoutsAttributeTypes reads the timeouts block's attributes from the schema, so a rebuilt
// value follows the schema instead of a hardcoded list. Returns nil if the block is missing.
func timeoutsAttributeTypes(s schema.Schema) map[string]attr.Type {
	// Indexing a missing key yields a nil Block, and calling Type() on it would panic.
	block, ok := s.Blocks["timeouts"]
	if !ok {
		return nil
	}
	t, ok := block.Type().(attr.TypeWithAttributeTypes)
	if !ok {
		return nil
	}
	return t.AttributeTypes()
}

func sdkv2TimeoutsToFramework(attrTypes map[string]attr.Type, t *sdkv2NamespaceTimeoutsV0) (timeouts.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	if attrTypes == nil {
		diags.AddError(moveStateErrSummary, "The timeouts block has no attribute types. This is a bug in the provider.")
		return timeouts.Value{}, diags
	}
	if t == nil {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}, nil
	}

	attrs := make(map[string]attr.Value, len(attrTypes))
	for k := range attrTypes {
		attrs[k] = types.StringNull()
	}
	if t.Delete != "" {
		attrs["delete"] = types.StringValue(t.Delete)
	}
	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return timeouts.Value{Object: obj}, diags
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// The identity schema is version 1, matching SDKv2's resourceIdentitySchemaNonNamespaced().
// Terraform asks for an upgrade whenever the identity stored in state carries a lower
// version, and the framework returns "Unable to Upgrade Resource Identity" unless the
// resource provides one. SDKv2 never needed this: its gRPC server upgrades identity
// generically, decoding the raw identity and re-coercing it against the current schema
// (helper/schema/grpc_provider.go:113).
//
// The attributes never changed, so this is a pass-through carrying the name across and
// restating the two constants.
//
// PriorSchema is deliberately not set. The framework decodes RawIdentity against it
// *before* calling this function and fails with "RawState had no JSON or flatmap data set"
// when there is nothing to decode — which is the normal case for a resource that was
// deferred and so never had an identity written. Reading RawIdentity directly lets that
// case be handled here instead, guarded the same way SDKv2 guards it ("no resource
// identity provided to upgrade", grpc_provider.go:146).
func (n *NamespaceV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return map[int64]resource.IdentityUpgrader{
		0: {
			IdentityUpgrader: func(ctx context.Context, req resource.UpgradeIdentityRequest, resp *resource.UpgradeIdentityResponse) {
				if resp.Identity == nil {
					return
				}

				// Pre-2.38.0 state has no identity. The framework rejects an empty
				// response, so return an object with every attribute null.
				// Framework v1.16.1+ lets Read populate it; setting any field here
				// would trigger "Unexpected Identity Change" when Read adds the name.
				if req.RawIdentity == nil || len(req.RawIdentity.JSON) == 0 {
					resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
						APIVersion: types.StringNull(),
						Kind:       types.StringNull(),
						Name:       types.StringNull(),
					})...)
					return
				}

				var prior struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(req.RawIdentity.JSON, &prior); err != nil {
					resp.Diagnostics.AddError(
						"Unable to upgrade kubernetes_namespace_v1 identity",
						fmt.Sprintf("Could not decode the stored identity: %s", err),
					)
					return
				}

				resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
					APIVersion: types.StringValue(namespaceAPIVersion),
					Kind:       types.StringValue(namespaceKind),
					Name:       types.StringValue(prior.Name),
				})...)
			},
		},
	}
}
