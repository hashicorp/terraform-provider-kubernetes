// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	limitRangeAPIVersion = "v1"
	limitRangeKind       = "LimitRange"
)

// Create implements [resource.Resource].
func (r *LimitRangeV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LimitRangeV1Model

	// Read from Plan, not Config: Config has not resolved computed values.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec, diags := expandLimitRangeSpec(ctx, plan.Spec, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	limitRange := corev1.LimitRange{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	tflog.Info(ctx, "Creating LimitRange", map[string]any{"namespace": metadata.Namespace, "name": metadata.Name})

	out, err := conn.CoreV1().LimitRanges(metadata.Namespace).Create(ctx, &limitRange, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating LimitRange", err.Error())
		return
	}
	tflog.Info(ctx, "Created LimitRange", map[string]any{"id": kubernetes.BuildId(out.ObjectMeta)})

	// Metadata server-assigned fields are taken from the response; labels/annotations
	// are left as the plan wrote them — filtering is Read's job.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	// spec.limit[*].default_request is Optional+Computed: the API server back-fills it
	// when omitted. The plan carries it as unknown, so we must resolve it from the
	// response rather than writing unknown back to state.
	flatSpec, flatDiags := flattenLimitRangeSpec(ctx, out.Spec)
	resp.Diagnostics.Append(flatDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Spec = flatSpec

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(limitRangeAPIVersion),
			Kind:       types.StringValue(limitRangeKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// Read implements [resource.Resource].
func (r *LimitRangeV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LimitRangeV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, metadataFilters, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	tflog.Info(ctx, "Reading LimitRange", map[string]any{"namespace": namespace, "name": name})
	out, err := conn.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			tflog.Info(ctx, "LimitRange no longer exists, removing from state",
				map[string]any{"namespace": namespace, "name": name})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to read LimitRange %q/%q", namespace, name), err.Error())
		return
	}

	// Prior state is the filtering reference. On import it is empty, which is the
	// correct baseline: nothing was declared, so nothing is exempt from filtering.
	metadata, diags := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, state.Metadata,
		metadataFilters.GetIgnoreAnnotations(), metadataFilters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec, diags := flattenLimitRangeSpec(ctx, out.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	state.Metadata = metadata
	state.Spec = spec

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(limitRangeAPIVersion),
			Kind:       types.StringValue(limitRangeKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// Update implements [resource.Resource].
func (r *LimitRangeV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LimitRangeV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state LimitRangeV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	// Build metadata patch ops from state vs plan.
	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)

	// Spec changes require a replace of the whole spec via JSON Patch.
	specOps, diags := specPatchOps(ctx, state.Spec, plan.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops = append(ops, specOps...)

	var out *corev1.LimitRange
	if len(ops) == 0 {
		// No managed keys changed — GET to resolve computed fields without a PATCH
		// permission requirement.
		out, err = conn.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read LimitRange during update", err.Error())
			return
		}
	} else {
		data, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal update JSON patch", err.Error())
			return
		}
		tflog.Info(ctx, "Updating LimitRange", map[string]any{"namespace": namespace, "name": name, "ops": string(data)})
		out, err = conn.CoreV1().LimitRanges(namespace).Patch(ctx, name, k8stypes.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update LimitRange", err.Error())
			return
		}
	}

	// Metadata server-assigned fields from response; labels/annotations from plan.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	// Resolve default_request (Optional+Computed) from the API response.
	flatSpec, flatDiags := flattenLimitRangeSpec(ctx, out.Spec)
	resp.Diagnostics.Append(flatDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Spec = flatSpec

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(limitRangeAPIVersion),
			Kind:       types.StringValue(limitRangeKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// Delete implements [resource.Resource].
func (r *LimitRangeV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LimitRangeV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	tflog.Info(ctx, "Deleting LimitRange", map[string]any{"namespace": namespace, "name": name})
	err = conn.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Already gone — treat as success so a resource removed out of band
			// can still be destroyed in Terraform.
			return
		}
		resp.Diagnostics.AddError("Failed to delete LimitRange", err.Error())
		return
	}
	tflog.Info(ctx, "Deleted LimitRange", map[string]any{"namespace": namespace, "name": name})
}

// ── expand / flatten helpers ──────────────────────────────────────────────────

// expandLimitRangeSpec converts the Framework spec model into a Kubernetes LimitRangeSpec.
// isNew mirrors the SDKv2 isNew flag that guards the Pod default_request validation.
func expandLimitRangeSpec(ctx context.Context, spec []LimitRangeSpecModel, isNew bool) (corev1.LimitRangeSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := corev1.LimitRangeSpec{}
	if len(spec) == 0 {
		return out, diags
	}

	items := make([]corev1.LimitRangeItem, len(spec[0].Limit))
	for i, l := range spec[0].Limit {
		item := corev1.LimitRangeItem{
			Type: corev1.LimitType(l.Type.ValueString()),
		}

		// default_request is forbidden for Pod limits even though the API echoes it back.
		// SDKv2 raised an error only on create (isNew), matching the original behaviour.
		if !l.DefaultRequest.IsNull() && !l.DefaultRequest.IsUnknown() {
			if item.Type == corev1.LimitTypePod && isNew {
				diags.AddError(
					"Invalid limit configuration",
					fmt.Sprintf("limit[%d].default_request cannot be set for Pod limit", i),
				)
				return out, diags
			}
			rl, d := expandResourceMap(ctx, l.DefaultRequest)
			diags.Append(d...)
			item.DefaultRequest = rl
		}

		if !l.Default.IsNull() && !l.Default.IsUnknown() {
			rl, d := expandResourceMap(ctx, l.Default)
			diags.Append(d...)
			item.Default = rl
		}
		if !l.Max.IsNull() && !l.Max.IsUnknown() {
			rl, d := expandResourceMap(ctx, l.Max)
			diags.Append(d...)
			item.Max = rl
		}
		if !l.MaxLimitRequestRatio.IsNull() && !l.MaxLimitRequestRatio.IsUnknown() {
			rl, d := expandResourceMap(ctx, l.MaxLimitRequestRatio)
			diags.Append(d...)
			item.MaxLimitRequestRatio = rl
		}
		if !l.Min.IsNull() && !l.Min.IsUnknown() {
			rl, d := expandResourceMap(ctx, l.Min)
			diags.Append(d...)
			item.Min = rl
		}

		if diags.HasError() {
			return out, diags
		}
		items[i] = item
	}
	out.Limits = items
	return out, diags
}

// expandResourceMap converts a types.Map of quantity strings into a corev1.ResourceList.
func expandResourceMap(ctx context.Context, m types.Map) (corev1.ResourceList, diag.Diagnostics) {
	var diags diag.Diagnostics
	raw := make(map[string]string, len(m.Elements()))
	diags.Append(m.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make(corev1.ResourceList, len(raw))
	for k, v := range raw {
		q, err := k8sresource.ParseQuantity(v)
		if err != nil {
			diags.AddError("Invalid resource quantity",
				fmt.Sprintf("Could not parse %q = %q: %s", k, v, err))
			return nil, diags
		}
		out[corev1.ResourceName(k)] = q
	}
	return out, diags
}

// flattenLimitRangeSpec converts the Kubernetes LimitRangeSpec into the Framework model.
// Empty spec (no limits) returns nil — matching SDKv2's return of []interface{}{} which
// the Framework treats as a null/absent block list.
func flattenLimitRangeSpec(ctx context.Context, in corev1.LimitRangeSpec) ([]LimitRangeSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(in.Limits) == 0 {
		return nil, diags
	}

	items := make([]LimitRangeItemModel, len(in.Limits))
	for i, l := range in.Limits {
		item := LimitRangeItemModel{
			Type: types.StringValue(string(l.Type)),
		}
		var d diag.Diagnostics

		item.Default, d = flattenResourceList(ctx, l.Default)
		diags.Append(d...)
		item.DefaultRequest, d = flattenResourceList(ctx, l.DefaultRequest)
		diags.Append(d...)
		item.Max, d = flattenResourceList(ctx, l.Max)
		diags.Append(d...)
		item.MaxLimitRequestRatio, d = flattenResourceList(ctx, l.MaxLimitRequestRatio)
		diags.Append(d...)
		item.Min, d = flattenResourceList(ctx, l.Min)
		diags.Append(d...)

		items[i] = item
	}
	return []LimitRangeSpecModel{{Limit: items}}, diags
}

// flattenResourceList converts a corev1.ResourceList to a types.Map of quantity strings.
// A nil/empty ResourceList returns a null map, preserving SDKv2's behaviour where an
// absent resource constraint stored nothing (null) rather than an empty map.
func flattenResourceList(ctx context.Context, l corev1.ResourceList) (types.Map, diag.Diagnostics) {
	if len(l) == 0 {
		return types.MapNull(types.StringType), nil
	}
	m := make(map[string]string, len(l))
	for k, v := range l {
		m[string(k)] = v.String()
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

// specPatchOps produces a JSON Patch replace operation when the spec has changed.
// Spec is treated as an atomic whole (matching SDKv2's ReplaceOperation on "/spec").
func specPatchOps(ctx context.Context, stateSpec, planSpec []LimitRangeSpecModel) (kubernetes.PatchOperations, diag.Diagnostics) {
	var diags diag.Diagnostics

	newSpec, d := expandLimitRangeSpec(ctx, planSpec, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	oldSpec, d := expandLimitRangeSpec(ctx, stateSpec, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	// Only emit the Replace op when the spec actually changed.
	// Comparing via JSON is cheaper than a deep reflect.DeepEqual on the API structs.
	oldJSON, err := encodeJSON(oldSpec)
	if err != nil {
		diags.AddError("Failed to encode prior spec", err.Error())
		return nil, diags
	}
	newJSON, err := encodeJSON(newSpec)
	if err != nil {
		diags.AddError("Failed to encode planned spec", err.Error())
		return nil, diags
	}
	if oldJSON == newJSON {
		return nil, diags
	}

	return kubernetes.PatchOperations{&kubernetes.ReplaceOperation{
		Path:  "/spec",
		Value: newSpec,
	}}, diags
}

func encodeJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
