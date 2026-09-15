// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// deprecatedNamespaceTypeName is the un-versioned alias that stays on the
	// SDKv2 server. It is the only source this resource accepts a move from.
	deprecatedNamespaceTypeName = "kubernetes_namespace"

	// deprecatedNamespaceSchemaVersion is the SDKv2 alias's schema version.
	// resourceKubernetesNamespaceV1 declares no SchemaVersion, so it is 0, and
	// the decoding below assumes exactly that shape.
	deprecatedNamespaceSchemaVersion = 0

	// providerAddressSuffix is matched against SourceProviderAddress by
	// suffix, deliberately ignoring the hostname: a full-string compare against
	// registry.terraform.io breaks every practitioner using a network mirror or
	// a private registry.
	providerAddressSuffix = "hashicorp/kubernetes"
)

// MoveState lets a practitioner retire the deprecated alias with a moved block
// instead of hand-editing state:
//
//	moved {
//	  from = kubernetes_namespace.example
//	  to   = kubernetes_namespace_v1.example
//	}
//
// This is the point of migrating the resource. Moving ACROSS type names is the
// only thing MoveState does, and SDKv2 cannot do it at all — which is why the
// alias-to-versioned rename has never been possible without state surgery.
//
// Note that this is a different mechanism from the UpgradeState in
// namespace_v1_upgrade.go, and the two do not compose: UpgradeState fires on the
// same type name at a lower stored schema version, MoveState on a type change.
// A moved state lands at the target's CURRENT schema version and no upgrader
// runs afterwards, so this mover has to apply the same value normalisation the
// v0 upgrader applies. Doing it in only one of the two places is how a moved
// resource ends up with a diff its owner cannot resolve.
func (r *NamespaceV1) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{{
		// SourceSchema is deliberately not set. Declaring it would mean
		// restating the entire SDKv2 schema here, and any drift between the two
		// silently yields null values rather than an error — the framework logs
		// the decode failure at DEBUG and calls the mover anyway. Decoding
		// SourceRawState into the explicit struct below fails loudly instead.
		StateMover: r.moveFromDeprecatedNamespace,
	}}
}

// namespaceSourceStateV0 is the state SDKv2 writes for kubernetes_namespace.
//
// Every field is a pointer or a map so that "absent" and "zero" stay
// distinguishable while decoding; collapsing them here would discard the exact
// information the normalisation below depends on.
type namespaceSourceStateV0 struct {
	ID       *string `json:"id"`
	Metadata []struct {
		Annotations     map[string]string `json:"annotations"`
		GenerateName    *string           `json:"generate_name"`
		Generation      *int64            `json:"generation"`
		Labels          map[string]string `json:"labels"`
		Name            *string           `json:"name"`
		ResourceVersion *string           `json:"resource_version"`
		UID             *string           `json:"uid"`
	} `json:"metadata"`
	Timeouts *struct {
		Delete *string `json:"delete"`
	} `json:"timeouts"`
	WaitForDefaultServiceAccount *bool `json:"wait_for_default_service_account"`
}

func (r *NamespaceV1) moveFromDeprecatedNamespace(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	// GATE FIRST, on all three of address, type and schema version.
	//
	// Returning with neither state nor diagnostics is how a mover declines: the
	// framework then tries the next mover, and only errors if none claim the
	// move. An under-gated mover instead claims moves it cannot correctly
	// perform — in particular a future source schema version whose JSON still
	// happens to unmarshal into the struct above would move silently, with
	// dropped or zeroed fields, and the next apply would patch the live
	// namespace to match that degraded state.
	if !strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
		return
	}
	if req.SourceTypeName != deprecatedNamespaceTypeName {
		return
	}
	if req.SourceSchemaVersion != deprecatedNamespaceSchemaVersion {
		return
	}
	if req.SourceRawState == nil {
		return
	}

	var src namespaceSourceStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &src); err != nil {
		resp.Diagnostics.AddError(
			"Unable to move kubernetes_namespace state",
			"The prior state of the source resource could not be read: "+err.Error(),
		)
		return
	}
	if len(src.Metadata) == 0 {
		resp.Diagnostics.AddError(
			"Unable to move kubernetes_namespace state",
			"The prior state of the source resource has no metadata block, so the namespace it refers to cannot be determined.",
		)
		return
	}

	srcMeta := src.Metadata[0]

	annotations, diags := stringMapValue(ctx, srcMeta.Annotations)
	resp.Diagnostics.Append(diags...)
	labels, diags := stringMapValue(ctx, srcMeta.Labels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	target := NamespaceV1Model{
		ID:                           stringValueOrNull(src.ID),
		WaitForDefaultServiceAccount: types.BoolValue(src.WaitForDefaultServiceAccount != nil && *src.WaitForDefaultServiceAccount),
		Timeouts:                     moveTimeouts(src),
		Metadata: []NamespaceMetadataModel{{
			// Same normalisation as the v0 state upgrader, for the same reason:
			// SDKv2 zero-fills the scalar leaves of a block, so a namespace
			// whose configuration omitted generate_name still carries "" here,
			// and the Framework plans null for it. The maps are normalised for
			// consistency; released 3.2.1 was measured writing null for them
			// already.
			Annotations:  nullIfEmptyMap(annotations),
			GenerateName: nullIfEmptyString(stringValueOrNull(srcMeta.GenerateName)),
			Labels:       nullIfEmptyMap(labels),

			Generation:      int64ValueOrNull(srcMeta.Generation),
			Name:            stringValueOrNull(srcMeta.Name),
			ResourceVersion: stringValueOrNull(srcMeta.ResourceVersion),
			UID:             stringValueOrNull(srcMeta.UID),
		}},
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Identity is carried across so the moved resource is importable and
	// participates in plan -generate-config-out immediately, rather than only
	// after the next refresh. TargetIdentity is nil when the client did not
	// negotiate identity support, and setting it then is an error.
	if resp.TargetIdentity != nil {
		resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, NamespaceV1IdentityModel{
			APIVersion: types.StringValue(namespaceAPIVersion),
			Kind:       types.StringValue(namespaceKind),
			Name:       stringValueOrNull(srcMeta.Name),
		})...)
	}
}

func moveTimeouts(src namespaceSourceStateV0) timeouts.Value {
	attrTypes := map[string]attr.Type{"delete": types.StringType}
	if src.Timeouts == nil {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}
	obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
		"delete": stringValueOrNull(src.Timeouts.Delete),
	})
	if diags.HasError() {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}
	return timeouts.Value{Object: obj}
}

func stringValueOrNull(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func int64ValueOrNull(i *int64) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*i)
}
