// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// Schema implements [resource.Resource].
func (n *NamespaceV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wait_for_default_service_account": schema.BoolAttribute{
				Description: "Terraform will wait for the default service account to be created.",
				Optional:    true,
				// Optional+Computed+Default is how the framework spells SDKv2's
				// `Default: false`. All three are required, or a config that omits
				// the field lands as null in state instead of false.
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.MetadataSchema("namespace", true),
			// SDKv2 declares only a Delete timeout, so only Delete is user-settable.
			// timeouts.BlockAll would add create/update/read and change the schema.
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Delete: true}),
		},
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (n *NamespaceV1) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		// Must match resourceIdentitySchemaNonNamespaced() in
		// kubernetes/resourceidentity.go. State written by the SDKv2 resource records
		// identity schema version 1; declaring 0 here asks Terraform to downgrade.
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
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

				// api_version and kind are constants for this resource, so only the name
				// has to come from the stored identity. It stays null when there is none:
				// a resource that was deferred never had an identity written. SDKv2 returns
				// an empty response in that case; the framework cannot, because it rejects
				// one with "Missing Upgraded Resource Identity", so a null name is the
				// closest equivalent. Read repopulates it once the object exists.
				name := types.StringNull()
				if req.RawIdentity != nil && len(req.RawIdentity.JSON) > 0 {
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
					if prior.Name != "" {
						name = types.StringValue(prior.Name)
					}
				}

				resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
					APIVersion: types.StringValue(namespaceAPIVersion),
					Kind:       types.StringValue(namespaceKind),
					Name:       name,
				})...)
			},
		},
	}
}
