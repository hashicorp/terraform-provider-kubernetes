// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestpatch

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubemanifest"
)

func (r *ManifestPatch) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan manifestPatchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	to, d := kubemanifest.OpTimeout(ctx, plan.Timeouts, "create", 0)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	ri, _, err := r.resolveTarget(&plan)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: resolve failed", err.Error())
		return
	}

	// Existence check: a patch targets an object it does not create. Refuse to run if the
	// target is missing (a Server-Side Apply would otherwise create a partial object).
	if _, err := ri.Get(ctx, plan.Name.ValueString(), metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			resp.Diagnostics.AddError("kubernetes_manifest_patch: target object not found",
				fmt.Sprintf("%s %q (namespace %q) does not exist. kubernetes_manifest_patch modifies an existing "+
					"object; create it first (e.g. with kubernetes_manifest_yaml) or add a `depends_on`.",
					plan.Kind.ValueString(), plan.Name.ValueString(), plan.Namespace.ValueString()))
			return
		}
		resp.Diagnostics.AddError("kubernetes_manifest_patch: read target failed", err.Error())
		return
	}

	live, err := r.apply(ctx, ri, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError(kubemanifest.ApplyErrDiag("kubernetes_manifest_patch", err))
		return
	}
	if err := setComputed(ctx, &plan, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManifestPatch) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state manifestPatchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ri, _, err := r.resolveTarget(&state)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: resolve failed", err.Error())
		return
	}
	live, err := ri.Get(ctx, state.Name.ValueString(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// The object we were patching is gone; the patch has nothing to manage.
		resp.Diagnostics.AddWarning("kubernetes_manifest_patch: target object no longer exists",
			"The patched object was deleted outside Terraform; removing the patch from state.")
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: read failed", err.Error())
		return
	}
	if err := setComputed(ctx, &state, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ManifestPatch) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan manifestPatchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	to, d := kubemanifest.OpTimeout(ctx, plan.Timeouts, "update", 0)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	ri, _, err := r.resolveTarget(&plan)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: resolve failed", err.Error())
		return
	}
	if _, err := ri.Get(ctx, plan.Name.ValueString(), metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			resp.Diagnostics.AddError("kubernetes_manifest_patch: target object not found",
				fmt.Sprintf("%s %q no longer exists; cannot apply the patch.", plan.Kind.ValueString(), plan.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("kubernetes_manifest_patch: read target failed", err.Error())
		return
	}

	live, err := r.apply(ctx, ri, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError(kubemanifest.ApplyErrDiag("kubernetes_manifest_patch", err))
		return
	}
	if err := setComputed(ctx, &plan, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManifestPatch) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state manifestPatchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// relinquish (default): do NOT touch the object. Simply forget the patch. This is why
	// destroying a patch never deletes parts of a cluster-managed object.
	if destroyBehaviorOf(&state) == destroyRelinquish {
		return
	}

	// remove_fields: prune the fields this manager owns by re-applying an identity-only
	// object (SSA removes fields we no longer declare). Validated to SSA-only in config.
	to, d := kubemanifest.OpTimeout(ctx, state.Timeouts, "delete", 0)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	ri, obj, err := r.resolveTarget(&state)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: resolve failed", err.Error())
		return
	}
	if _, err := ri.Get(ctx, state.Name.ValueString(), metav1.GetOptions{}); apierrors.IsNotFound(err) {
		return // target already gone; nothing to prune
	}

	body, err := obj.MarshalJSON() // identity-only object → prunes our owned fields
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: encode failed", err.Error())
		return
	}
	force := true
	_, err = ri.Patch(ctx, state.Name.ValueString(), apitypes.ApplyPatchType, body,
		metav1.PatchOptions{FieldManager: fieldManagerOf(&state), Force: &force})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: remove_fields failed", err.Error())
		return
	}
}

// ModifyPlan previews owned-field drift for the SSA patch path (dry-run SSA → project
// owned → compare to prior state). Target-identity changes are handled by RequiresReplace
// plan modifiers, not here.
func (r *ManifestPatch) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return // destroy or create
	}
	var plan, state manifestPatchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// patch_json has no SSA ownership → no owned-field drift preview.
	if plan.usesJSONPatch() || plan.Patch.IsNull() || plan.Patch.IsUnknown() {
		return
	}

	ri, _, err := r.resolveTarget(&plan)
	if err != nil {
		return // cluster unreachable at plan → degrade gracefully
	}
	dry, err := r.apply(ctx, ri, &plan, true)
	if err != nil {
		if kubemanifest.IsImmutableErr(err) {
			resp.Diagnostics.AddError(
				"kubernetes_manifest_patch: change requires replacement",
				fmt.Sprintf("The planned patch modifies a field Kubernetes does not allow to be updated in "+
					"place:\n\n%s\n\nThis field cannot be patched; it can only be changed by replacing the object "+
					"(outside this resource).", err.Error()),
			)
		}
		return
	}
	projected, err := kubemanifest.ProjectOwned(dry, fieldManagerOf(&plan), ignoreList(ctx, &plan))
	if err != nil {
		return
	}
	if projected != state.OwnedManifest.ValueString() {
		plan.OwnedManifest = types.StringValue(projected)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// ImportState hydrates identity/computed fields from a key=value ID + a live Get. patch/
// patch_json are user config and cannot be reconstructed; the user must add matching config.
func (r *ManifestPatch) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := parsePatchID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: invalid import ID", err.Error())
		return
	}

	var state manifestPatchModel
	state.APIVersion = types.StringValue(id.APIVersion)
	state.Kind = types.StringValue(id.Kind)
	state.Name = types.StringValue(id.Name)
	state.Namespace = types.StringValue(id.Namespace)
	fm := id.FieldManager
	if fm == "" {
		fm = defaultFieldManager
	}
	state.FieldManager = types.StringValue(fm)
	state.ForceConflicts = types.BoolNull()
	state.DestroyBehavior = types.StringValue(destroyRelinquish)
	state.PatchType = types.StringNull()
	state.PatchJSON = types.StringNull()
	state.Patch = types.DynamicNull()
	state.IgnoreFields = types.ListNull(types.StringType)
	state.Timeouts = timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
		"create": types.StringType, "read": types.StringType, "update": types.StringType, "delete": types.StringType,
	})}

	ri, _, err := r.resolveTarget(&state)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: resolve failed", err.Error())
		return
	}
	live, err := ri.Get(ctx, id.Name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: import get failed",
			fmt.Sprintf("could not read %s/%s: %s", id.Namespace, id.Name, err))
		return
	}
	if err := setComputed(ctx, &state, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_patch: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
