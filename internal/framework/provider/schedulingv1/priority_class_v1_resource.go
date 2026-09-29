// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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

// UpgradeIdentity handles version-0 identity from provider versions before 2.38.0
// (e.g. v2.37.1) where identity_schema_version was 0 or identity was absent.
//
// Without this, any object created by provider 2.37.x or older fails its first plan with
// "Unable to Upgrade Resource Identity": identity shipped in 2.38.0, so older state carries
// identity_schema_version 0 and no identity, and Terraform asks for an upgrade whenever the
// stored version differs from the declared one. SDKv2 answers that generically in its gRPC
// server; the framework requires each resource to supply it.
func (r *PriorityClassV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity("PriorityClass", "scheduling.k8s.io/v1")
}

func (r *PriorityClassV1) MoveState(_ context.Context) []resource.StateMover {
	return moveStateHandlers()
}

// ConfigValidators enforces that exactly one metadata block is present at plan
// time, and that user-defined PriorityClasses do not exceed the Kubernetes
// maximum of 1,000,000,000.
//
// The value cap is applied as a ConfigValidator (not an attribute validator) so
// that it can inspect the name: built-in system classes (system-cluster-critical,
// system-node-critical) use values above the cap and must be importable.
func (r *PriorityClassV1) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&userDefinedValueValidator{},
	}
}

// userDefinedValueValidator enforces the Kubernetes HighestUserDefinablePriority
// cap (1,000,000,000) for user-defined PriorityClasses. Built-in system classes
// whose names start with "system-" are exempt because they legitimately use values
// above the cap (system-cluster-critical = 2,000,000,000).
//
// The lower bound is math.MinInt32: the Kubernetes API field is int32, so values
// below −2,147,483,648 are rejected by the API anyway.
type userDefinedValueValidator struct{}

func (v *userDefinedValueValidator) Description(_ context.Context) string {
	return "user-defined priority class value must be between -2147483648 and 1000000000"
}

func (v *userDefinedValueValidator) MarkdownDescription(_ context.Context) string {
	return "user-defined priority class `value` must be between `-2147483648` and `1000000000`"
}

func (v *userDefinedValueValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config PriorityClassModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cannot validate if value is unknown (e.g. computed from another resource).
	if config.Value.IsUnknown() || config.Value.IsNull() {
		return
	}

	val := config.Value.ValueInt64()

	// Lower bound: int32 minimum — the Kubernetes API uses an int32 field.
	if val < math.MinInt32 {
		resp.Diagnostics.AddError(
			"Invalid Priority Class Value",
			fmt.Sprintf("value must be at least %d (int32 minimum), got: %d", math.MinInt32, val),
		)
		return
	}

	// Upper bound: only enforced for user-defined classes.
	// System classes (system-cluster-critical, system-node-critical) exceed the cap
	// and must remain importable. Skip the check when name starts with "system-".
	if val > 1_000_000_000 {
		// Determine the effective name: use name if set, fall back to generate_name prefix.
		name := ""
		if len(config.Metadata) > 0 {
			if !config.Metadata[0].Name.IsNull() && !config.Metadata[0].Name.IsUnknown() {
				name = config.Metadata[0].Name.ValueString()
			} else if !config.Metadata[0].GenerateName.IsNull() && !config.Metadata[0].GenerateName.IsUnknown() {
				name = config.Metadata[0].GenerateName.ValueString()
			}
		}
		if !strings.HasPrefix(name, "system-") {
			resp.Diagnostics.AddError(
				"Invalid Priority Class Value",
				fmt.Sprintf(
					"value must be at most 1000000000 for user-defined priority classes, got: %d. "+
						"Values above 1,000,000,000 are reserved for system-critical classes "+
						"(names starting with \"system-\").",
					val,
				),
			)
		}
	}
}
