// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package manifestpatch implements kubernetes_manifest_patch (RFC-012): a declarative
// Server-Side Apply resource that owns a SUBSET of fields on a pre-existing object it
// does not create or delete. It is the generic form of kubernetes_labels/annotations/
// *_data, built on the shared kubemanifest engine.
package manifestpatch

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	defaultFieldManager = "terraform-patch"

	destroyRelinquish   = "relinquish"
	destroyRemoveFields = "remove_fields"

	patchTypeApply     = "apply"
	patchTypeMerge     = "merge"
	patchTypeStrategic = "strategic"
	patchTypeJSON      = "json"
)

var (
	_ resource.Resource                     = (*ManifestPatch)(nil)
	_ resource.ResourceWithConfigure        = (*ManifestPatch)(nil)
	_ resource.ResourceWithImportState      = (*ManifestPatch)(nil)
	_ resource.ResourceWithModifyPlan       = (*ManifestPatch)(nil)
	_ resource.ResourceWithConfigValidators = (*ManifestPatch)(nil)
	_ resource.ResourceWithValidateConfig   = (*ManifestPatch)(nil)
)

// ManifestPatch owns a subset of fields on an existing Kubernetes object via SSA.
type ManifestPatch struct {
	// SDKv2Meta returns the SDKv2-initialized provider meta (shared client).
	SDKv2Meta func() any
}

// NewManifestPatch is the resource factory registered on the provider.
func NewManifestPatch() resource.Resource {
	return &ManifestPatch{}
}

func (r *ManifestPatch) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest_patch"
}

func (r *ManifestPatch) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *ManifestPatch) clients() kubernetes.KubeClientsets {
	return r.SDKv2Meta().(kubernetes.KubeClientsets)
}

// manifestPatchModel is the tfsdk state model. It stores the target identity, the
// user's patch, the field_manager, and the owned-field projection (drift anchor) —
// never the whole target object.
type manifestPatchModel struct {
	APIVersion      types.String  `tfsdk:"api_version"`
	Kind            types.String  `tfsdk:"kind"`
	Name            types.String  `tfsdk:"name"`
	Namespace       types.String  `tfsdk:"namespace"`
	Patch           types.Dynamic `tfsdk:"patch"`
	PatchJSON       types.String  `tfsdk:"patch_json"`
	PatchType       types.String  `tfsdk:"patch_type"`
	IgnoreFields    types.List    `tfsdk:"ignore_fields"`
	FieldManager    types.String  `tfsdk:"field_manager"`
	ForceConflicts  types.Bool    `tfsdk:"force_conflicts"`
	DestroyBehavior types.String  `tfsdk:"destroy_behavior"`

	ID            types.String `tfsdk:"id"`
	OwnedManifest types.String `tfsdk:"owned_manifest"`
	ObjectExists  types.Bool   `tfsdk:"object_exists"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// usesJSONPatch reports whether the escape-hatch patch_json path is in use.
func (m *manifestPatchModel) usesJSONPatch() bool {
	return !m.PatchJSON.IsNull() && !m.PatchJSON.IsUnknown() && m.PatchJSON.ValueString() != ""
}
