// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IdentitySchemaVersion matches SDKv2 identity since provider 2.38.0.
// Lowering it would reject existing state as an unsupported downgrade.
const IdentitySchemaVersion = 1

// ResourceIdentity is the identity model for a cluster-scoped object, matching the schema
// returned by IdentitySchema.
type ResourceIdentity struct {
	Name       types.String `tfsdk:"name"`
	Kind       types.String `tfsdk:"kind"`
	APIVersion types.String `tfsdk:"api_version"`
}

// NamespacedResourceIdentity adds namespace to ResourceIdentity without nesting its wire shape.
type NamespacedResourceIdentity struct {
	ResourceIdentity
	Namespace types.String `tfsdk:"namespace"`
}

// IdentitySchema returns the identity schema for a cluster-scoped object. It reproduces
// resourceIdentitySchemaNonNamespaced() from kubernetes/resourceidentity.go.
func IdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Version:    IdentitySchemaVersion,
		Attributes: identityAttributes(),
	}
}

// NamespacedIdentitySchema mirrors SDKv2 resourceIdentitySchemaNamespaced.
func NamespacedIdentitySchema() identityschema.Schema {
	attributes := identityAttributes()

	attributes["namespace"] = identityschema.StringAttribute{
		OptionalForImport: true,
	}

	return identityschema.Schema{
		Version:    IdentitySchemaVersion,
		Attributes: attributes,
	}
}

func identityAttributes() map[string]identityschema.Attribute {
	return map[string]identityschema.Attribute{
		"name":        identityschema.StringAttribute{RequiredForImport: true},
		"kind":        identityschema.StringAttribute{RequiredForImport: true},
		"api_version": identityschema.StringAttribute{RequiredForImport: true},
	}
}

// UpgradeIdentity handles version-0 cluster-scoped identity, including absent identity
// from provider versions before 2.38.0. Framework requires this explicit upgrader;
// SDKv2 handled it generically.
func UpgradeIdentity(kind, apiVersion string) map[int64]resource.IdentityUpgrader {
	return upgradeIdentity(kind, apiVersion, false)
}

// UpgradeNamespacedIdentity is UpgradeIdentity for a namespaced resource: it carries the
// stored namespace across as well as the name.
func UpgradeNamespacedIdentity(kind, apiVersion string) map[int64]resource.IdentityUpgrader {
	return upgradeIdentity(kind, apiVersion, true)
}

func upgradeIdentity(kind, apiVersion string, namespaced bool) map[int64]resource.IdentityUpgrader {
	return map[int64]resource.IdentityUpgrader{
		0: {
			// No PriorSchema: Framework would reject absent raw identity before calling us.
			IdentityUpgrader: func(ctx context.Context, req resource.UpgradeIdentityRequest, resp *resource.UpgradeIdentityResponse) {
				if resp.Identity == nil {
					return
				}

				// An absent identity must be fully null so Read can populate it later.
				identity := NamespacedResourceIdentity{
					ResourceIdentity: ResourceIdentity{
						Name:       types.StringNull(),
						Kind:       types.StringNull(),
						APIVersion: types.StringNull(),
					},
					Namespace: types.StringNull(),
				}
				if req.RawIdentity != nil && len(req.RawIdentity.JSON) > 0 {
					var prior struct {
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					}
					if err := json.Unmarshal(req.RawIdentity.JSON, &prior); err != nil {
						resp.Diagnostics.AddError(
							"Unable to upgrade resource identity",
							fmt.Sprintf("Could not decode the stored identity: %s", err),
						)
						return
					}
					identity.Name = types.StringValue(prior.Name)
					identity.Kind = types.StringValue(kind)
					identity.APIVersion = types.StringValue(apiVersion)
					identity.Namespace = types.StringValue(prior.Namespace)
				}

				if namespaced {
					resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
					return
				}
				resp.Diagnostics.Append(resp.Identity.Set(ctx, identity.ResourceIdentity)...)
			},
		},
	}
}
