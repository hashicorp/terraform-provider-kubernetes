// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UpgradeState carries SDKv2-written state forward to schema version 1.
//
// The stored TYPE does not change — `metadata` was and remains a list of
// objects with the same seven attributes — only the value conventions do.
//
// Measured against released provider 3.2.1 on kind v1.34.0 (do the measurement;
// this is not derivable by argument): a namespace created with only a name
// persists `generate_name: ""` but `annotations: null` and `labels: null`.
// SDKv2 zero-fills the SCALAR leaves of a block it writes, while an empty
// TypeMap becomes null rather than `{}`. So `generate_name` is the field that
// actually needs converting here — the Framework plans null for it, and left at
// "" every existing namespace would plan `"" -> null` on its first plan after
// upgrading, on configuration nobody touched.
//
// The maps are normalised anyway. They cost nothing when already null, and
// SDKv2's map handling is not uniform across resources: the pod measurement
// recorded in the domain pack's inbox found `{}`.
//
// Because the type is unchanged, PriorSchema can be the current schema and the
// conversion works on decoded values rather than on raw JSON. That is
// deliberately not the pod_v1 approach: raw-JSON surgery is only necessary when
// a prior version's type genuinely differs, and it gives up compiler and
// decoder checking for no benefit here.
//
// This is the one place that nulls an empty value unconditionally. Everywhere
// else the prior known value is preserved, because everywhere else there is one
// to consult. SDKv2 stored an explicitly configured `annotations = {}` and an
// omitted one identically, so at upgrade the two are indistinguishable; the
// omitted case is the one that must stay diff-free, and the explicit case
// resolves itself with a single no-op-to-the-API update.
func (r *NamespaceV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	priorSchema := namespaceV1Schema(ctx)

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				if req.State == nil {
					resp.Diagnostics.AddError(
						"Missing prior state",
						"The prior state was empty, so it cannot be upgraded.",
					)
					return
				}

				var state NamespaceV1Model
				resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
				if resp.Diagnostics.HasError() {
					return
				}

				for i := range state.Metadata {
					state.Metadata[i].Annotations = nullIfEmptyMap(state.Metadata[i].Annotations)
					state.Metadata[i].Labels = nullIfEmptyMap(state.Metadata[i].Labels)
					state.Metadata[i].GenerateName = nullIfEmptyString(state.Metadata[i].GenerateName)
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			},
		},
	}
}

func nullIfEmptyMap(v types.Map) types.Map {
	if isKnownEmptyMap(v) {
		return types.MapNull(types.StringType)
	}
	return v
}

func nullIfEmptyString(v types.String) types.String {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() == "" {
		return types.StringNull()
	}
	return v
}
