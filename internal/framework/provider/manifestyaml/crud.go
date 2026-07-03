// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apitypes "k8s.io/apimachinery/pkg/types"
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

	force := m.ForceConflicts.ValueBool() || preview // dry-run (preview) forces so drift detection isn't masked by conflicts
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

// setComputed writes the identity/status computed fields from a live object.
func setComputed(m *manifestYAMLModel, obj *unstructured.Unstructured) {
	m.ID = types.StringValue(buildID(obj))
	m.APIVersion = types.StringValue(obj.GetAPIVersion())
	m.Kind = types.StringValue(obj.GetKind())
	m.Name = types.StringValue(obj.GetName())
	if ns := obj.GetNamespace(); ns != "" {
		m.Namespace = types.StringValue(ns)
	} else {
		m.Namespace = types.StringValue("") // cluster-scoped: known empty string, not null
	}
	m.UID = types.StringValue(string(obj.GetUID()))
	m.ResourceVersion = types.StringValue(obj.GetResourceVersion())
}

// objectIdentity builds a lookup stub from the model's computed identity fields,
// falling back to yaml_body. This lets Read work even when yaml_body is unavailable
// (e.g. immediately after import).
func objectIdentity(m *manifestYAMLModel) (*unstructured.Unstructured, error) {
	if m.APIVersion.ValueString() != "" && m.Kind.ValueString() != "" && m.Name.ValueString() != "" {
		u := &unstructured.Unstructured{}
		u.SetAPIVersion(m.APIVersion.ValueString())
		u.SetKind(m.Kind.ValueString())
		u.SetName(m.Name.ValueString())
		u.SetNamespace(m.Namespace.ValueString())
		return u, nil
	}
	return decodeYAML(m.YamlBody.ValueString())
}

func (r *ManifestYAML) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan manifestYAMLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	to, d := opTimeout(ctx, plan.Timeouts, "create", plan.Wait)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	out, err := r.apply(ctx, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError(applyErrDiag(err))
		return
	}
	setComputed(&plan, out)
	if err := setLiveManifest(ctx, &plan, out); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	if err := setStatus(&plan, out); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: status read failed", err.Error())
		return
	}
	if err := r.waitForReady(ctx, out, plan.Wait); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: wait failed", err.Error())
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

	obj, err := objectIdentity(&state)
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
	if err := setStatus(&state, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: status read failed", err.Error())
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

	to, d := opTimeout(ctx, plan.Timeouts, "update", plan.Wait)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	out, err := r.apply(ctx, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError(applyErrDiag(err))
		return
	}
	setComputed(&plan, out)
	if err := setLiveManifest(ctx, &plan, out); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	if err := setStatus(&plan, out); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: status read failed", err.Error())
		return
	}
	if err := r.waitForReady(ctx, out, plan.Wait); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: wait failed", err.Error())
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

	// Semantic-equality (RFC-011 §6.4): if the planned YAML normalizes to the same
	// document as state (only whitespace/comments/key-order differ), suppress the diff
	// by keeping the prior value.
	if n1, e1 := normalizeYAML(plan.YamlBody.ValueString()); e1 == nil {
		if n2, e2 := normalizeYAML(state.YamlBody.ValueString()); e2 == nil && n1 == n2 {
			plan.YamlBody = state.YamlBody
			resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		}
	}

	// buildID encodes apiVersion/kind/namespace/name — the object's identity.
	if buildID(planObj) != buildID(stateObj) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("yaml_body"))
		return // replacement supersedes drift projection
	}

	// force_replace_on: user-declared immutable paths (e.g. a StatefulSet's
	// spec.volumeClaimTemplates). Changing any of them replaces the object instead of
	// attempting an update that Kubernetes would reject. Deterministic and offline-safe.
	for _, p := range replaceOnList(ctx, &plan) {
		if pathChanged(planObj.Object, stateObj.Object, p) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("yaml_body"))
			return
		}
	}

	// Owned-field drift (RFC-011 §6.3): dry-run SSA the planned object, project the
	// owned fields, and compare to prior state. If they differ, the object has drifted
	// (or the config changed) → surface it via live_manifest and mark the server-assigned
	// computed fields unknown so the corrective apply stays consistent.
	// Degrades gracefully: if the cluster is unreachable at plan, leave computed as-is.
	dry, err := r.apply(ctx, &plan, true)
	if err != nil {
		// A rejected update to an immutable field means the change needs replacement.
		// Guide the user to force_replace_on rather than letting apply fail with a raw 422.
		if isImmutableErr(err) {
			resp.Diagnostics.AddError(
				"kubernetes_manifest_yaml: change requires replacement",
				fmt.Sprintf("The planned change modifies a field that Kubernetes does not allow to be "+
					"updated in place:\n\n%s\n\nAdd the changed field path(s) to `force_replace_on` so Terraform "+
					"replaces the object instead of updating it. For workloads that must keep their Pods/PVCs across "+
					"the replacement (e.g. StatefulSets), also set `delete { propagation_policy = \"Orphan\" }`.",
					err.Error()),
			)
		}
		return
	}
	projected, err := projectOwned(dry, fieldManagerOf(&plan), ignoreList(ctx, &plan))
	if err != nil {
		return
	}
	if projected != state.LiveManifest.ValueString() {
		plan.LiveManifest = types.StringValue(projected)
		// These change when the corrective apply runs; mark unknown to avoid
		// "inconsistent result after apply".
		plan.ResourceVersion = types.StringUnknown()
		plan.Status = types.StringUnknown()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *ManifestYAML) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state manifestYAMLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	to, d := opTimeout(ctx, state.Timeouts, "delete", nil)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

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

	ri := resourceInterface(dyn, gvr, namespaced, obj.GetNamespace())
	err = ri.Delete(ctx, obj.GetName(), deleteOptions(state.Delete))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return // already gone
		}
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: delete failed", err.Error())
		return
	}

	// Block until the object is actually gone so Terraform does not report success while
	// finalizers still hold it. Orphan/Background deletes return quickly; Foreground waits
	// for dependents. Bounded by the delete timeout.
	if err := waitForDeleted(ctx, ri, obj.GetName()); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: delete did not complete", err.Error())
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
	state.ForceReplaceOn = types.ListNull(types.StringType)
	state.YamlBody = types.StringNull() // user must supply matching config
	// A freshly-built model needs a correctly-typed (not zero-value) timeouts object,
	// otherwise state.Set fails type conversion on the timeouts attribute.
	state.Timeouts = timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	})}
	setComputed(&state, live)
	if err := setLiveManifest(ctx, &state, live); err != nil {
		resp.Diagnostics.AddError("kubernetes_manifest_yaml: owned-field projection failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
