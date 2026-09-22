// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MoveState implements resource.ResourceWithMoveState. It enables a
// practitioner to rename an existing kubernetes_cluster_role_binding resource
// (the deprecated SDKv2 alias) to kubernetes_cluster_role_binding_v1 (this
// Framework resource) using a moved { } configuration block:
//
//	moved {
//	  from = kubernetes_cluster_role_binding.example
//	  to   = kubernetes_cluster_role_binding_v1.example
//	}
//
// The two resources share an identical schema layout (both use list blocks for
// metadata, role_ref and subject with no namespace at the resource level) so
// the migration is a direct state copy with identity population.
//
// This functionality requires Terraform 1.8 or later.
func (r *ClusterRoleBinding) MoveState(_ context.Context) []resource.StateMover {
	// Obtain the resource schema so the framework can decode SourceState for us.
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	return []resource.StateMover{
		{
			// Providing SourceSchema lets the framework populate
			// req.SourceState so we can use typed Get() instead of raw JSON.
			// State conversion errors are only DEBUG-logged; we still guard
			// with an explicit type-name check inside the mover.
			SourceSchema: &schemaResp.Schema,
			StateMover:   moveFromClusterRoleBinding,
		},
	}
}

// moveFromClusterRoleBinding is the StateMover implementation.
//
// It accepts state originating from the SDKv2 resource type
// kubernetes_cluster_role_binding (the deprecated bare-name alias). Because
// the schemas are structurally identical, decoding then re-encoding the state
// is sufficient — no field-level transformation is required.
//
// The function guards against unrelated move requests by checking
// SourceTypeName; if the request is not for the expected source type, the
// response is left empty (framework considers the mover skipped).
func moveFromClusterRoleBinding(
	ctx context.Context,
	req resource.MoveStateRequest,
	resp *resource.MoveStateResponse,
) {
	// Only handle moves from the deprecated non-v1 alias.
	// We match the suffix so the check is robust across provider FQDNs.
	// The bare name ends in "kubernetes_cluster_role_binding" but NOT
	// "kubernetes_cluster_role_binding_v1" — check the v1 suffix first.
	if strings.HasSuffix(req.SourceTypeName, "kubernetes_cluster_role_binding_v1") ||
		!strings.HasSuffix(req.SourceTypeName, "kubernetes_cluster_role_binding") {
		// Either already the v1 type, or something entirely unrelated — skip.
		return
	}

	// Decode the source state into our shared model. req.SourceState is
	// populated because we provided SourceSchema in the StateMover above.
	var src ClusterRoleBindingModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &src)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The source and target schemas are structurally identical: the SDKv2
	// kubernetes_cluster_role_binding uses the same TypeList blocks for
	// metadata, role_ref and subject, and has no top-level namespace field
	// because ClusterRoleBinding is cluster-scoped.  A direct copy suffices.
	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &src)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Populate the Framework resource identity.  The SDKv2 provider stored
	// these in a separate identity document (resourceIdentitySchemaNonNamespaced).
	// We derive them from the canonical constants rather than trying to parse
	// the old SDKv2 identity JSON, so they are always correct.
	name := src.ID.ValueString()
	resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, ClusterRoleBindingIdentityModel{
		APIVersion: types.StringValue(clusterRoleBindingAPIVersion),
		Kind:       types.StringValue(clusterRoleBindingKind),
		Name:       types.StringValue(name),
	})...)
}
