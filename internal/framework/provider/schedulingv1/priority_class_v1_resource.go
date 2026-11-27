// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	v1 "k8s.io/api/core/v1"
)

var (
	_ resource.Resource                     = (*PriorityClassV1)(nil) // compile-time check: PriorityClassV1 must implement all interface methods
	_ resource.ResourceWithConfigure        = (*PriorityClassV1)(nil)
	_ resource.ResourceWithImportState      = (*PriorityClassV1)(nil)
	_ resource.ResourceWithIdentity         = (*PriorityClassV1)(nil)
	_ resource.ResourceWithMoveState        = (*PriorityClassV1)(nil)
	_ resource.ResourceWithUpgradeIdentity  = (*PriorityClassV1)(nil)
	_ resource.ResourceWithConfigValidators = (*PriorityClassV1)(nil)
)

const (
	priorityClassAPIVersion = "scheduling.k8s.io/v1"
	priorityClassKind       = "PriorityClass"

	// highestUserDefinablePriority is the largest value the API server accepts for a
	// PriorityClass whose name does not start with "system-".
	highestUserDefinablePriority = 1_000_000_000
)

type PriorityClassV1 struct {
	SDKv2Meta func() any
}

func NewPriorityClassV1() resource.Resource {
	return &PriorityClassV1{}
}

func (r *PriorityClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_priority_class_v1"
}

func (r *PriorityClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *PriorityClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// No SDKv2 release declared resource identity for this type, 3.5.0 included, so every state
// SDKv2 wrote carries identity_schema_version 0 and no identity. Terraform asks for an
// upgrade whenever the stored version differs from the declared one, even when nothing is
// stored. SDKv2 answers that generically in its gRPC server; the framework requires each
// resource to supply it. common.UpgradeIdentity returns an all-null identity, which Read
// then fills in.
func (r *PriorityClassV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(priorityClassKind, priorityClassAPIVersion)
}

// The source provider address is HOSTNAME/NAMESPACE/TYPE, and the hostname is ignored: it is
// the public registry for most users and a mirror or private registry for others. The leading
// slash anchors the match to a whole segment, so a fork such as nothashicorp/kubernetes does
// not match.
const sdkv2ProviderAddressSuffix = "/hashicorp/kubernetes"

const moveStateErrSummary = "Unable to move kubernetes_priority_class state"

// MoveState implements [resource.ResourceWithMoveState] for
//
//	moved {
//	  from = kubernetes_priority_class.example
//	  to   = kubernetes_priority_class_v1.example
//	}
//
// the supported route off the deprecated alias, which stays on SDKv2. Terraform allows a
// cross-type move only when the destination provider implements this.
//
// The source is decoded from raw JSON into a frozen struct rather than through a
// SourceSchema, so it does not follow this resource's schema as that changes. A future SDKv2
// schema version needs its own struct and a SourceSchemaVersion branch, not edits to this one.
func (r *PriorityClassV1) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			// No SourceSchema: the framework would decode against it before calling us.
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "kubernetes_priority_class" ||
					req.SourceSchemaVersion != 0 ||
					!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
					// Skipping means returning no state and no diagnostics, and the framework
					// then fails the move without saying why. Leave a reason in the log.
					tflog.Debug(ctx, "MoveState: not a kubernetes_priority_class v0 source, skipping", map[string]any{
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

				var prior sdkv2PriorityClassStateV0
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
					MetadataBase: common.MetadataBase{
						Name:            types.StringValue(m.Name),
						Generation:      types.Int64Value(m.Generation),
						ResourceVersion: types.StringValue(m.ResourceVersion),
						UID:             types.StringValue(m.UID),
					},
				}
				meta.Labels, diags = common.SDKv2MapToFramework(ctx, m.Labels)
				resp.Diagnostics.Append(diags...)
				meta.Annotations, diags = common.SDKv2MapToFramework(ctx, m.Annotations)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				// SDKv2 writes "" for an unset generate_name, where the framework needs null.
				// Left as "", a plan with -refresh=false compares "" against null and
				// RequiresReplace recreates the priority class.
				meta.GenerateName = types.StringNull()
				if m.GenerateName != "" {
					meta.GenerateName = types.StringValue(m.GenerateName)
				}

				// SDKv2 stores its default, so "" only appears in hand-edited state.
				preemptionPolicy := prior.PreemptionPolicy
				if preemptionPolicy == "" {
					preemptionPolicy = string(v1.PreemptLowerPriority)
				}

				state := PriorityClassModel{
					ID:               types.StringValue(prior.ID),
					Metadata:         []common.MetadataModel{meta},
					Value:            types.Int64Value(prior.Value),
					Description:      types.StringValue(prior.Description),
					GlobalDefault:    types.BoolValue(prior.GlobalDefault),
					PreemptionPolicy: types.StringValue(preemptionPolicy),
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &state)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				// Build the identity rather than decoding req.SourceIdentity: the SDKv2 alias
				// never had one, api_version and kind are constants, and the name is in the
				// state just moved.
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, priorityClassIdentity(m.Name))...)
			},
		},
	}
}

func priorityClassIdentity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		APIVersion: types.StringValue(priorityClassAPIVersion),
		Kind:       types.StringValue(priorityClassKind),
		Name:       types.StringValue(name),
	}
}

// ConfigValidators enforces that user-defined PriorityClasses do not exceed the
// Kubernetes maximum of 1,000,000,000 and all classes fit within the int32 range.
//
// The value cap is applied as a ConfigValidator (not an attribute validator) so
// that it can inspect the name: built-in system classes (system-cluster-critical,
// system-node-critical) use values above 1,000,000,000 and must remain valid and importable.
func (r *PriorityClassV1) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&userDefinedValueValidator{},
	}
}

// userDefinedValueValidator enforces the Kubernetes PriorityClass value constraints:
//   - All PriorityClasses must fit in an int32 (math.MinInt32 to math.MaxInt32)
//     to prevent integer overflow when sending to the Kubernetes API.
//   - User-defined PriorityClasses (names not starting with "system-") are capped at
//     HighestUserDefinablePriority (1,000,000,000).
//   - Built-in system classes (names starting with "system-") may use values up to
//     math.MaxInt32 (e.g. system-node-critical = 2,000,001,000).
type userDefinedValueValidator struct{}

func (v *userDefinedValueValidator) Description(_ context.Context) string {
	return "priority class value must be between -2147483648 and 2147483647 (and at most 1000000000 for user-defined classes)"
}

func (v *userDefinedValueValidator) MarkdownDescription(_ context.Context) string {
	return "priority class `value` must be between `-2147483648` and `2147483647` (and at most `1000000000` for user-defined classes)"
}

func (v *userDefinedValueValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// Retrieve the value attribute directly so int32 bounds are checked even if
	// the metadata block or collection is unknown (e.g. from dynamic blocks).
	var value types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value"), &value)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cannot validate if value is unknown or null.
	if value.IsUnknown() || value.IsNull() {
		return
	}

	val := value.ValueInt64()

	// Bounds check: value must fit in an int32 because the Kubernetes API field is int32.
	if val < math.MinInt32 {
		resp.Diagnostics.AddError(
			"Invalid Priority Class Value",
			fmt.Sprintf("value must be at least %d (int32 minimum), got: %d", math.MinInt32, val),
		)
		return
	}
	if val > math.MaxInt32 {
		resp.Diagnostics.AddError(
			"Invalid Priority Class Value",
			fmt.Sprintf("value must be at most %d (int32 maximum), got: %d", math.MaxInt32, val),
		)
		return
	}

	if val <= highestUserDefinablePriority {
		return
	}

	// Above the user cap, the answer depends on the name: system classes
	// (system-cluster-critical, system-node-critical) exceed it and must remain
	// valid and importable.
	//
	// A null metadata list decodes to an empty slice. An unknown one (a dynamic block
	// over values not known until apply) cannot be decoded at all. Both leave the name
	// unknown here, so defer to the API server.
	var metadata []common.MetadataModel
	if diags := req.Config.GetAttribute(ctx, path.Root("metadata"), &metadata); diags.HasError() || len(metadata) == 0 {
		return
	}

	m := metadata[0]
	// If either name or generate_name is unknown, defer validation because
	// it could resolve to a "system-" prefix at apply time.
	if m.Name.IsUnknown() || m.GenerateName.IsUnknown() {
		return
	}

	name := m.Name.ValueString()
	if m.Name.IsNull() {
		name = m.GenerateName.ValueString()
	}

	if !strings.HasPrefix(name, "system-") {
		resp.Diagnostics.AddError(
			"Invalid Priority Class Value",
			fmt.Sprintf(
				"value must be at most %d for user-defined priority classes, got: %d. "+
					"Values above %d are reserved for system-critical classes "+
					"(names starting with \"system-\").",
				highestUserDefinablePriority, val, highestUserDefinablePriority,
			),
		)
	}
}
