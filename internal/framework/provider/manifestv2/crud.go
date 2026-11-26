// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

const (
	defaultTimeout = 10 * time.Minute
	// acceptedKey stores, in private state, the manifest fields whose live
	// value differed right after an apply.
	acceptedKey = "accepted"
)

type privateState interface {
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// prepareApply reads what an apply needs from the planned values.
func prepareApply(ctx context.Context, plan model) (map[string]any, identity, []expectation, error) {
	manifest, declared, known, err := plan.declared()
	if err == nil && (!known || manifest == nil) {
		err = fmt.Errorf("the manifest is not fully known")
	}
	if err != nil {
		return nil, declared, nil, err
	}
	expected, err := expectations(ctx, plan.Wait, declared)
	return manifest, declared, expected, err
}

// recordAppliedState completes the state after an apply. The object attribute
// is read here only when the plan left it open, which it does for a new
// object: ModifyPlan explains why an update keeps it.
func recordAppliedState(ctx context.Context, plan *model, id identity, manifest map[string]any, applied *unstructured.Unstructured, private privateState) diag.Diagnostics {
	plan.setIdentity(id)
	plan.UID = types.StringValue(string(applied.GetUID()))
	plan.Drift = types.ListValueMust(types.StringType, nil)
	manager, _ := plan.manager()
	_, accepted := liveChanges(manifest, applied, manager, plan.computedFields(), nil)
	encoded, _ := json.Marshal(accepted)
	diags := private.SetKey(ctx, acceptedKey, encoded)
	if plan.Object.IsUnknown() {
		diags.Append(observe(ctx, plan, applied)...)
	}
	return diags
}

// observe sets the object attribute: the live object without the metadata
// that only tracks writes.
func observe(ctx context.Context, state *model, live *unstructured.Unstructured) diag.Diagnostics {
	var diags diag.Diagnostics
	obj := live.DeepCopy()
	for _, field := range []string{"managedFields", "resourceVersion", "selfLink"} {
		unstructured.RemoveNestedField(obj.Object, "metadata", field)
	}
	value, err := fromJSON(ctx, obj.Object)
	if err != nil {
		diags.AddError("Cannot read object", err.Error())
		return diags
	}
	state.Object = types.DynamicValue(value)
	return diags
}

// waitForReadiness waits for the object and, when the plan left the object
// value open, reads it again. The state was already written, so a wait that fails leaves
// the object in state.
func waitForReadiness(ctx context.Context, client dynamic.ResourceInterface, id identity, expected []expectation, plan *model, open bool, state *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(expected) == 0 {
		return diags
	}
	if err := await(ctx, client, id.Name, expected); err != nil {
		diags.AddError("Object applied but not ready", fmt.Sprintf("%s: %s", id, err))
		return diags
	}
	if !open {
		return diags
	}
	live, err := client.Get(ctx, id.Name, metav1.GetOptions{})
	if err != nil {
		diags.AddError("Cannot read object", fmt.Sprintf("%s: %s", id, err))
		return diags
	}
	diags.Append(observe(ctx, plan, live)...)
	diags.Append(state.Set(ctx, plan)...)
	return diags
}

func applyError(id identity, err error) diag.Diagnostic {
	detail := err.Error()
	switch {
	case strings.Contains(detail, "uid mismatch") || strings.Contains(detail, "metadata.uid"):
		return diag.NewErrorDiagnostic("Object deleted or replaced outside Terraform", fmt.Sprintf("%s is no longer the object this resource "+
			"manages: %s\n\nRun terraform plan again. Terraform will plan to create the object, or to update the one that replaced it.", id, detail))
	case apierrors.IsConflict(err):
		return diag.NewErrorDiagnostic("Field ownership conflict", fmt.Sprintf("%s: %s\n\n"+
			"Another field manager owns these fields with a different value. Remove them from the manifest to leave them "+
			"to that manager, or set force_conflicts = true in the field_manager block to take them over.", id, detail))
	case strings.Contains(detail, "immutable") || strings.Contains(detail, "may not change once set"):
		detail += "\n\nAn immutable field can only change by replacing the object, for example with terraform apply -replace=ADDRESS."
	}
	return diag.NewErrorDiagnostic("Cannot apply manifest", fmt.Sprintf("%s: %s", id, detail))
}

func (r *ManifestV2) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	manifest, id, expected, err := prepareApply(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Objects of one configuration are applied in parallel, so the namespace
	// or the CustomResourceDefinition this one needs may arrive a moment later.
	var client dynamic.ResourceInterface
	err = retryWhile(ctx, awaitsDependency, func() (err error) {
		client, id, err = r.client(id, false)
		return err
	})
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	if _, err := client.Get(ctx, id.Name, metav1.GetOptions{}); err == nil {
		resp.Diagnostics.AddError("Object already exists", fmt.Sprintf("%s already exists in the cluster. "+
			"To manage it with this resource, import it with the ID %q.", id, importID(id)))
		return
	} else if !apierrors.IsNotFound(err) {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	manager, force := plan.manager()
	var applied *unstructured.Unstructured
	err = retryWhile(ctx, awaitsDependency, func() (err error) {
		applied, err = apply(ctx, client, applicable(manifest, id), manager, force)
		return err
	})
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	open := plan.Object.IsUnknown()
	resp.Diagnostics.Append(recordAppliedState(ctx, &plan, id, manifest, applied, resp.Private)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, plan.resourceIdentity())...)
	resp.Diagnostics.Append(waitForReadiness(ctx, client, id, expected, &plan, open, &resp.State)...)
}

func (r *ManifestV2) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Terraform before 1.12 sends no identity on refresh, but Framework
	// requires one even when Read removes the resource.
	if resp.Identity != nil && resp.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, state.resourceIdentity())...)
	}
	client, id, err := r.client(state.identity(), true)
	var live *unstructured.Unstructured
	if err == nil {
		live, err = client.Get(ctx, id.Name, metav1.GetOptions{})
	}
	if err == nil && id.APIVersion != state.APIVersion.ValueString() {
		resp.Diagnostics.AddWarning("API version no longer served", fmt.Sprintf("The cluster no longer serves %s in API version %s. "+
			"The object is now read as %s. Change apiVersion in the manifest to a version the cluster serves.",
			id.Kind, state.APIVersion.ValueString(), id.APIVersion))
	}
	if _, unknownKind := err.(unknownKindError); unknownKind || apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read object", fmt.Sprintf("%s: %s", id, err))
		return
	}

	state.setIdentity(id)
	state.UID = types.StringValue(string(live.GetUID()))
	resp.Diagnostics.Append(observe(ctx, &state, live)...)
	manager, force := state.manager()
	// described is set when the manifest in state no longer describes the
	// object: after an import, or when another client changed it.
	var described map[string]any
	var paths []string
	if state.Manifest.IsNull() && state.YAMLBody.IsNull() {
		described = exported(live, manager)
	} else {
		manifest, _, _, err := state.declared()
		if err != nil || manifest == nil {
			resp.Diagnostics.AddError("Invalid manifest in state", fmt.Sprint(err))
			return
		}
		accepted := map[string]string{}
		if encoded, _ := req.Private.GetKey(ctx, acceptedKey); encoded != nil {
			_ = json.Unmarshal(encoded, &accepted)
		}
		changed, drifted := liveChanges(manifest, live, manager, state.computedFields(), accepted)
		var conflicts []string
		for path, found := range drifted {
			paths = append(paths, path)
			if found != absent {
				conflicts = append(conflicts, path)
			}
		}
		sort.Strings(paths)
		sort.Strings(conflicts)
		if len(conflicts) > 0 && !force {
			resp.Diagnostics.AddWarning("Object changed by another field manager", fmt.Sprintf("%s: another client changed %s. "+
				"Applying the manifest will fail with a field ownership conflict unless these fields are removed from it "+
				"or field_manager.force_conflicts is true.", id, strings.Join(conflicts, ", ")))
		}
		// A manifest given as an object shows the live values in the plan,
		// where lifecycle.ignore_changes can also leave them alone. One given
		// as text lists the fields in drift: its values may be sensitive.
		if state.YAMLBody.IsNull() {
			paths = nil
			if len(drifted) > 0 {
				described = changed
			}
		}
	}
	drift, diags := types.ListValueFrom(ctx, types.StringType, append([]string{}, paths...))
	resp.Diagnostics.Append(diags...)
	state.Drift = drift
	if described != nil {
		value, err := fromJSON(ctx, described)
		if err != nil {
			resp.Diagnostics.AddError("Cannot read object", fmt.Sprintf("%s: %s", id, err))
			return
		}
		state.Manifest = types.DynamicValue(value)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, state.resourceIdentity())...)
}

func (r *ManifestV2) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	manifest, declared, expected, err := prepareApply(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The object stays the one in state. Only its API version follows the manifest.
	id := state.identity()
	id.APIVersion = declared.APIVersion
	client, id, err := r.client(id, false)
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	manager, force := plan.manager()
	// Nothing is written until the live object is known to be the one in
	// state: it may have been deleted or replaced since the plan.
	uid := state.UID.ValueString()
	live, err := client.Get(ctx, id.Name, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		err = fmt.Errorf("uid mismatch: the object no longer exists")
	case err == nil && uid != "" && string(live.GetUID()) != uid:
		err = fmt.Errorf("uid mismatch: the object in the cluster has UID %s, the state has %s", live.GetUID(), uid)
	}
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	if err := adopt(ctx, client, live, manager); err != nil {
		resp.Diagnostics.AddError("Cannot take over the object's fields", fmt.Sprintf("%s: %s", id, err))
		return
	}
	// The apply names the UID too, for a replacement between here and there.
	desired := applicable(leavingComputed(manifest, live, manager, plan.computedFields()), id)
	if uid != "" {
		desired.SetUID(k8stypes.UID(uid))
	}
	applied, err := apply(ctx, client, desired, manager, force)
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	if previous, _ := state.manager(); previous != manager {
		if err := release(ctx, client, id, previous); err != nil {
			resp.Diagnostics.AddWarning("Previous field manager not released", fmt.Sprintf(
				"%s is applied as %q, but field manager %q still owns its fields: %s", id, manager, previous, err))
		}
	}
	open := plan.Object.IsUnknown()
	resp.Diagnostics.Append(recordAppliedState(ctx, &plan, id, manifest, applied, resp.Private)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, plan.resourceIdentity())...)
	resp.Diagnostics.Append(waitForReadiness(ctx, client, id, expected, &plan, open, &resp.State)...)
}

func (r *ManifestV2) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := state.Timeouts.Delete(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	uid := k8stypes.UID(state.UID.ValueString())
	client, id, err := r.client(state.identity(), true)
	if err == nil {
		// Naming the UID keeps the delete from removing another object that
		// took the name since the plan.
		options := metav1.DeleteOptions{}
		if uid != "" {
			options.Preconditions = &metav1.Preconditions{UID: &uid}
		}
		err = client.Delete(ctx, id.Name, options)
	}
	// Gone already: the object, its kind in every version, or its namespace.
	if _, unknownKind := err.(unknownKindError); unknownKind || apierrors.IsNotFound(err) {
		return
	}
	// The API answers a UID that is not the object's with a conflict: the name
	// belongs to another object now, which is not this resource's to delete.
	if apierrors.IsConflict(err) && uid != "" {
		live, getErr := client.Get(ctx, id.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(getErr) || getErr == nil && live.GetUID() != uid {
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot delete object", fmt.Sprintf("%s: %s", id, err))
		return
	}
	var finalizers []string
	err = wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		live, err := client.Get(ctx, id.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		finalizers = live.GetFinalizers()
		// Another UID is a new object under the same name; ours is gone.
		return string(live.GetUID()) != state.UID.ValueString(), nil
	})
	if err != nil {
		detail := fmt.Sprintf("%s was not removed within %s: %s.", id, timeout, err)
		if len(finalizers) > 0 {
			detail += fmt.Sprintf(" It is waiting for its finalizers: %s.", strings.Join(finalizers, ", "))
		}
		resp.Diagnostics.AddError("Object not deleted", detail)
	}
}

func importID(id identity) string {
	if id.Namespace == "" {
		return fmt.Sprintf("apiVersion=%s,kind=%s,name=%s", id.APIVersion, id.Kind, id.Name)
	}
	return fmt.Sprintf("apiVersion=%s,kind=%s,namespace=%s,name=%s", id.APIVersion, id.Kind, id.Namespace, id.Name)
}

// exported describes an imported object the way a manifest would. An object
// this resource applied before is described by the fields it owns, which is
// the manifest it was applied from. Any other object is described in full,
// without status and the bookkeeping the API server and kubectl add.
func exported(live *unstructured.Unstructured, manager string) map[string]any {
	obj := applicable(live.Object, identity{Namespace: live.GetNamespace()})
	if owned := managedFields(live, appliedBy(manager)); len(owned) > 0 {
		obj.Object, _ = project(obj.Object, owned).(map[string]any)
		obj.SetAPIVersion(live.GetAPIVersion())
		obj.SetKind(live.GetKind())
		obj.SetName(live.GetName())
		obj.SetNamespace(live.GetNamespace())
		return obj.Object
	}
	annotations := obj.GetAnnotations()
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	obj.SetAnnotations(annotations)
	return obj.Object
}
