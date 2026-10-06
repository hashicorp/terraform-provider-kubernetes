// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

const (
	storageClassKind       = "StorageClass"
	storageClassAPIVersion = "storage.k8s.io/v1"
)

// StorageClassV1 is the Plugin Framework resource for kubernetes_storage_class_v1.
type StorageClassV1 struct {
	SDKv2Meta func() any
}

var (
	_ resource.Resource                    = (*StorageClassV1)(nil)
	_ resource.ResourceWithConfigure       = (*StorageClassV1)(nil)
	_ resource.ResourceWithImportState     = (*StorageClassV1)(nil)
	_ resource.ResourceWithIdentity        = (*StorageClassV1)(nil)
	_ resource.ResourceWithMoveState       = (*StorageClassV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*StorageClassV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*StorageClassV1)(nil)
)

func NewStorageClassV1() resource.Resource {
	return &StorageClassV1{}
}

func (r *StorageClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_class_v1"
}

func (r *StorageClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

// IdentitySchema implements [resource.ResourceWithIdentity].
// Uses common.IdentitySchema which matches SDKv2 resourceIdentitySchemaNonNamespaced at Version 1.
func (r *StorageClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// Without this, any object created by provider 2.37.x or older fails its first plan with
// "Unable to Upgrade Resource Identity": identity shipped in 2.38.0, so older state carries
// identity_schema_version 0 and no identity, and Terraform asks for an upgrade whenever the
// stored version differs from the declared one. SDKv2 answers that generically in its gRPC
// server; the framework requires each resource to supply it.
func (r *StorageClassV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(storageClassKind, storageClassAPIVersion)
}

func (r *StorageClassV1) MoveState(_ context.Context) []resource.StateMover {
	return moveStateHandlers()
}

func (r *StorageClassV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If creating or destroying, no plan modification needed
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan StorageClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Force replacement whenever allowed_topologies differs at any depth.
	// StorageClass.allowedTopologies is immutable in Kubernetes; any change
	// (term count, expression key, or values set) requires a destroy+create.
	// Comparing only slice length (the prior check) missed mutations to inner
	// match_label_expressions entries that kept the term count the same.
	if !reflect.DeepEqual(state.AllowedTopologies, plan.AllowedTopologies) {
		resp.RequiresReplace.Append(path.Root("allowed_topologies"))
	}
}
