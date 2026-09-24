// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
