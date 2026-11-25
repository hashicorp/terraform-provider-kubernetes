// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	storagev1api "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// SDKv2 declared no timeouts, so its delete wait fell through to the SDK's 20 minute default.
const csiDriverDeleteTimeout = 20 * time.Minute

// Create implements [resource.Resource].
func (r *CSIDriverV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CSIDriverV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, _, diags := r.clientset()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata, diags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	spec, diags := expandCSIDriverV1Spec(ctx, plan.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := conn.StorageV1().CSIDrivers().Create(ctx, &storagev1api.CSIDriver{ObjectMeta: metadata, Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating CSIDriver", err.Error())
		return
	}

	plan.ID = types.StringValue(out.Name)
	common.PopulateMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	resp.Diagnostics.Append(setDefaultedVolumeLifecycleModes(ctx, plan.Spec, out.Spec)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, csiDriverIdentity(out.Name))...)
}

// Read implements [resource.Resource].
func (r *CSIDriverV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CSIDriverV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, filters, diags := r.clientset()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.ID.ValueString()
	out, err := conn.StorageV1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			tflog.Info(ctx, "CSIDriver no longer exists, removing from state", map[string]any{"name": name})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading CSIDriver %q", name), err.Error())
		return
	}

	// Prior state decides which server-added keys were declared; on import it is empty.
	metadata, diags := common.FlattenMetadata(ctx, out.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	spec, diags := flattenCSIDriverV1Spec(ctx, out.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(out.Name)
	state.Metadata = metadata
	state.Spec = spec
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, csiDriverIdentity(out.Name))...)
}

// Update implements [resource.Resource].
func (r *CSIDriverV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CSIDriverV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, _, diags := r.clientset()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0], plan.Metadata[0])
	specOps, diags := csiDriverV1SpecPatchOps(ctx, state.Spec, plan.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops = append(ops, specOps...)

	name := plan.ID.ValueString()
	var out *storagev1api.CSIDriver
	var err error
	if len(ops) == 0 {
		// Nothing for Kubernetes to change, but resource_version and generation must
		// still come from the live object.
		out, err = conn.StorageV1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
	} else {
		var patch []byte
		patch, err = ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error marshalling CSIDriver patch", err.Error())
			return
		}
		out, err = conn.StorageV1().CSIDrivers().Patch(ctx, name, k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error updating CSIDriver %q", name), err.Error())
		return
	}

	common.PopulateMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	resp.Diagnostics.Append(setDefaultedVolumeLifecycleModes(ctx, plan.Spec, out.Spec)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, csiDriverIdentity(out.Name))...)
}

// Delete implements [resource.Resource].
func (r *CSIDriverV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CSIDriverV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, _, diags := r.clientset()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.ID.ValueString()
	err := conn.StorageV1().CSIDrivers().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error deleting CSIDriver %q", name), err.Error())
		return
	}

	err = retry.RetryContext(ctx, csiDriverDeleteTimeout, func() *retry.RetryError {
		_, err := conn.StorageV1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("CSIDriver %q still exists", name))
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error waiting for CSIDriver %q to be deleted", name), err.Error())
	}
}

// clientset is resolved per request rather than in Configure, because the mux configures
// the SDKv2 provider independently and its meta may not be ready when Configure runs.
func (r *CSIDriverV1) clientset() (*k8sclient.Clientset, kubernetes.MetadataFilters, diag.Diagnostics) {
	clients, filters, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	if diags.HasError() {
		return nil, nil, diags
	}
	conn, err := clients.MainClientset()
	if err != nil {
		diags.AddError("Kubernetes client error", err.Error())
		return nil, nil, diags
	}
	return conn, filters, diags
}

func expandCSIDriverV1Spec(ctx context.Context, in []CSIDriverV1SpecModel) (storagev1api.CSIDriverSpec, diag.Diagnostics) {
	if len(in) == 0 {
		return storagev1api.CSIDriverSpec{}, nil
	}
	modes, diags := expandVolumeLifecycleModes(ctx, in[0].VolumeLifecycleModes)
	return storagev1api.CSIDriverSpec{
		AttachRequired:       in[0].AttachRequired.ValueBoolPointer(),
		PodInfoOnMount:       in[0].PodInfoOnMount.ValueBoolPointer(),
		VolumeLifecycleModes: modes,
	}, diags
}

// expandVolumeLifecycleModes returns nil for an omitted list, leaving the API to apply its default.
func expandVolumeLifecycleModes(ctx context.Context, in types.List) ([]storagev1api.VolumeLifecycleMode, diag.Diagnostics) {
	if in.IsNull() || in.IsUnknown() {
		return nil, nil
	}
	var values []string
	diags := in.ElementsAs(ctx, &values, false)
	modes := make([]storagev1api.VolumeLifecycleMode, 0, len(values))
	for _, v := range values {
		modes = append(modes, storagev1api.VolumeLifecycleMode(v))
	}
	return modes, diags
}

// flattenCSIDriverV1Spec always returns one spec element, as SDKv2's Read did, so an
// omitted spec block shows the API's defaults instead of hiding them.
func flattenCSIDriverV1Spec(ctx context.Context, in storagev1api.CSIDriverSpec) ([]CSIDriverV1SpecModel, diag.Diagnostics) {
	modes := make([]string, 0, len(in.VolumeLifecycleModes))
	for _, mode := range in.VolumeLifecycleModes {
		modes = append(modes, string(mode))
	}
	modeList, diags := types.ListValueFrom(ctx, types.StringType, modes)
	return []CSIDriverV1SpecModel{{
		AttachRequired:       types.BoolPointerValue(in.AttachRequired),
		PodInfoOnMount:       types.BoolValue(ptr.Deref(in.PodInfoOnMount, false)),
		VolumeLifecycleModes: modeList,
	}}, diags
}

// setDefaultedVolumeLifecycleModes resolves volume_lifecycle_modes when the config omitted
// it: the plan holds unknown, and state after apply must hold the value the API defaulted.
func setDefaultedVolumeLifecycleModes(ctx context.Context, spec []CSIDriverV1SpecModel, out storagev1api.CSIDriverSpec) diag.Diagnostics {
	if len(spec) == 0 || !spec[0].VolumeLifecycleModes.IsUnknown() {
		return nil
	}
	flattened, diags := flattenCSIDriverV1Spec(ctx, out)
	spec[0].VolumeLifecycleModes = flattened[0].VolumeLifecycleModes
	return diags
}

// csiDriverV1SpecPatchOps patches only the spec fields that changed. attach_required is
// absent because it forces replacement, and a removed spec block sends nothing, since
// patching the API's defaults back in would only fail on the immutable fields.
func csiDriverV1SpecPatchOps(ctx context.Context, state, plan []CSIDriverV1SpecModel) (kubernetes.PatchOperations, diag.Diagnostics) {
	ops := kubernetes.PatchOperations{}
	if len(plan) == 0 {
		return ops, nil
	}
	var prior CSIDriverV1SpecModel
	if len(state) > 0 {
		prior = state[0]
	}

	if !plan[0].PodInfoOnMount.Equal(prior.PodInfoOnMount) {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/spec/podInfoOnMount",
			Value: plan[0].PodInfoOnMount.ValueBool(),
		})
	}

	modes := plan[0].VolumeLifecycleModes
	if modes.IsUnknown() || modes.Equal(prior.VolumeLifecycleModes) {
		return ops, nil
	}
	expanded, diags := expandVolumeLifecycleModes(ctx, modes)
	ops = append(ops, &kubernetes.ReplaceOperation{
		Path:  "/spec/volumeLifecycleModes",
		Value: expanded,
	})
	return ops, diags
}
