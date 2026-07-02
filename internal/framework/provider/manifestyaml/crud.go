// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// apply performs a Server-Side Apply of the object described by the model's yaml_body.
// When preview is true the apply is a server dry-run (used for plan/diff).
func (r *ManifestYAML) apply(ctx context.Context, m *manifestYAMLModel, preview bool) (*unstructured.Unstructured, error) {
	obj, err := decodeYAML(m.YamlBody.ValueString())
	if err != nil {
		return nil, err
	}
	gvr, namespaced, err := resolveGVR(r.clients(), obj)
	if err != nil {
		return nil, err
	}
	dyn, err := r.clients().DynamicClient()
	if err != nil {
		return nil, err
	}
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}

	force := m.ForceConflicts.ValueBool()
	opts := metav1.PatchOptions{
		FieldManager: m.FieldManager.ValueString(),
		Force:        &force,
	}
	if preview {
		opts.DryRun = []string{metav1.DryRunAll}
	}

	ri := resourceInterface(dyn, gvr, namespaced, obj.GetNamespace())
	return ri.Patch(ctx, obj.GetName(), apitypes.ApplyPatchType, data, opts)
}

func resourceInterface(dyn dynamic.Interface, gvr k8sschema.GroupVersionResource, namespaced bool, ns string) dynamic.ResourceInterface {
	if namespaced {
		return dyn.Resource(gvr).Namespace(ns)
	}
	return dyn.Resource(gvr)
}

// setComputed writes the identity/status computed fields from a live object.
func setComputed(m *manifestYAMLModel, obj *unstructured.Unstructured) {
	m.ID = types.StringValue(buildID(obj))
	m.APIVersion = types.StringValue(obj.GetAPIVersion())
	m.Kind = types.StringValue(obj.GetKind())
	m.Name = types.StringValue(obj.GetName())
	if ns := obj.GetNamespace(); ns != "" {
		m.Namespace = types.StringValue(ns)
	} else {
		m.Namespace = types.StringNull()
	}
	m.UID = types.StringValue(string(obj.GetUID()))
	m.ResourceVersion = types.StringValue(obj.GetResourceVersion())
}

func (r *ManifestYAML) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan manifestYAMLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.apply(ctx, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: apply failed", err.Error())
		return
	}
	setComputed(&plan, out)
	if err := setLiveManifest(ctx, &plan, out); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManifestYAML) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state manifestYAMLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := decodeYAML(state.YamlBody.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: invalid stored manifest", err.Error())
		return
	}
	gvr, namespaced, err := resolveGVR(r.clients(), obj)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: resolve failed", err.Error())
		return
	}
	dyn, err := r.clients().DynamicClient()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	live, err := resourceInterface(dyn, gvr, namespaced, obj.GetNamespace()).
		Get(ctx, obj.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx) // drift: object deleted out-of-band
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: read failed", err.Error())
		return
	}

	setComputed(&state, live)
	if err := setLiveManifest(ctx, &state, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ManifestYAML) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan manifestYAMLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.apply(ctx, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: apply failed", err.Error())
		return
	}
	setComputed(&plan, out)
	if err := setLiveManifest(ctx, &plan, out); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// ModifyPlan forces replacement when the object's identity (apiVersion, kind,
// namespace, or name) changes. Without this, Update would apply the new object
// and orphan the old one. Identity change ⇒ destroy (old state) + create (new).
func (r *ManifestYAML) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy plan (no planned object) or create (no prior state): nothing to compare.
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var plan, state manifestYAMLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// If yaml_body is unknown at plan time, defer the identity check to apply.
	if plan.YamlBody.IsUnknown() || state.YamlBody.IsUnknown() {
		return
	}

	planObj, err := decodeYAML(plan.YamlBody.ValueString())
	if err != nil {
		return // invalid YAML is surfaced during apply, not here
	}
	stateObj, err := decodeYAML(state.YamlBody.ValueString())
	if err != nil {
		return
	}

	// buildID encodes apiVersion/kind/namespace/name — the object's identity.
	if buildID(planObj) != buildID(stateObj) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("yaml_body"))
		return // replacement supersedes drift projection
	}

	// Owned-field drift (RFC-011 §6.3): dry-run SSA the planned object, project the
	// owned fields, and set live_manifest in the plan. If it differs from prior state,
	// the diff on live_manifest surfaces drift and triggers an in-place update.
	// Degrades gracefully: if the cluster is unreachable at plan, leave it computed.
	dry, err := r.apply(ctx, &plan, true)
	if err != nil {
		return
	}
	if err := setLiveManifest(ctx, &plan, dry); err != nil {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *ManifestYAML) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state manifestYAMLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := decodeYAML(state.YamlBody.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: invalid stored manifest", err.Error())
		return
	}
	gvr, namespaced, err := resolveGVR(r.clients(), obj)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: resolve failed", err.Error())
		return
	}
	dyn, err := r.clients().DynamicClient()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	err = resourceInterface(dyn, gvr, namespaced, obj.GetNamespace()).
		Delete(ctx, obj.GetName(), deleteOptions(state.Delete))
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: delete failed", err.Error())
		return
	}
}

func deleteOptions(d *deleteModel) metav1.DeleteOptions {
	opts := metav1.DeleteOptions{}
	if d == nil {
		return opts
	}
	if !d.PropagationPolicy.IsNull() && d.PropagationPolicy.ValueString() != "" {
		p := metav1.DeletionPropagation(d.PropagationPolicy.ValueString())
		opts.PropagationPolicy = &p
	}
	if !d.GracePeriodSeconds.IsNull() {
		g := d.GracePeriodSeconds.ValueInt64()
		opts.GracePeriodSeconds = &g
	}
	return opts
}

// ImportState hydrates identity/computed fields from a key=value ID and a live Get.
// Note: yaml_body is Required and cannot be reconstructed; the user must add matching
// config (or use `terraform plan -generate-config-out`). See RFC-011 §2.1.
func (r *ManifestYAML) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: invalid import ID", err.Error())
		return
	}
	stub := unstructuredForImport(id)
	gvr, namespaced, err := resolveGVR(r.clients(), stub)
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: resolve failed", err.Error())
		return
	}
	dyn, err := r.clients().DynamicClient()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}
	live, err := resourceInterface(dyn, gvr, namespaced, id.Namespace).
		Get(ctx, id.Name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: import get failed",
			fmt.Sprintf("could not read %s/%s: %s", id.Namespace, id.Name, err))
		return
	}

	var state manifestYAMLModel
	state.FieldManager = types.StringValue("terraform")
	state.ForceConflicts = types.BoolNull()
	state.IgnoreFields = types.ListNull(types.StringType)
	state.YamlBody = types.StringNull() // user must supply matching config
	setComputed(&state, live)
	if err := setLiveManifest(ctx, &state, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
