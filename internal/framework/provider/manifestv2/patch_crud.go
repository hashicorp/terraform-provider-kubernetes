// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// arrival is how long a new patch waits for its object.
const arrival = time.Minute

// errReplaced is the error of a write to an object that is not the one it
// was meant for: another object has taken the name.
var errReplaced = errors.New("the object was deleted and created again")

// gone reports a write the API server refused because the object with the UID
// it named is not there: deleted, or replaced by another under the same name.
// An apply to a missing object is answered with a conflict, one to another
// object with an invalid metadata.uid. The error is read by its status: its
// text holds the name of the object, which may spell anything.
func gone(err error) (deleted, replaced bool) {
	var status apierrors.APIStatus
	switch {
	case errors.Is(err, errReplaced):
		return false, true
	case !errors.As(err, &status):
		return false, false
	case apierrors.IsNotFound(err):
		return true, false
	case apierrors.IsConflict(err):
		// No object name holds a space.
		return strings.Contains(status.Status().Message, "no existing object was found"), false
	}
	return false, uidInvalid(err)
}

func notFound(id identity, waited time.Duration) diag.Diagnostic {
	missing := "does not exist"
	if waited > 0 {
		missing = fmt.Sprintf("still does not exist after %s", waited.Round(time.Second))
	}
	return diag.NewErrorDiagnostic("Object not found", fmt.Sprintf("%s %s. kubernetes_manifest_patch applies fields to an object "+
		"that exists and never creates one. Make this resource depend on the resource that creates the object, or check its name and namespace.", id, missing))
}

// replaced reports an object that is no longer the one in state.
func replaced(id identity, detail string) diag.Diagnostic {
	return diag.NewErrorDiagnostic("Object replaced outside Terraform", fmt.Sprintf("%s was deleted and created again since the last refresh: %s\n\n"+
		"Run terraform plan again. Terraform will plan to apply the fields to the new object.", id, detail))
}

// existing reads the object a new patch applies to. One that is missing may
// be on its way: a controller creates the default ServiceAccount of a new
// namespace, and an operator the objects of a custom resource, a moment after
// the resource this one depends on is complete.
// It returns how long it waited for an object that did not come.
func existing(ctx context.Context, client dynamic.ResourceInterface, name string) (*unstructured.Unstructured, time.Duration, error) {
	start, delay := time.Now(), time.Second
	var missing error
	for {
		live, err := client.Get(ctx, name, metav1.GetOptions{})
		switch {
		case apierrors.IsNotFound(err):
			missing = err
		case missing != nil && ctx.Err() != nil:
			// The create timeout ended the wait.
			return nil, time.Since(start), missing
		default:
			return live, 0, err
		}
		if time.Since(start) > arrival {
			return nil, time.Since(start), missing
		}
		select {
		case <-ctx.Done():
			return nil, time.Since(start), missing
		case <-time.After(delay):
		}
		delay = min(2*delay, 4*time.Second)
	}
}

func (r *ManifestPatch) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan patchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	manifest, id, expected, err := prepareApply(ctx, plan.manifest())
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The kind may come from a CustomResourceDefinition applied in parallel.
	var client dynamic.ResourceInterface
	namespace := id.Namespace
	err = retryWhile(ctx, id, func() (err error) {
		client, id, err = r.engine.client(id, false)
		return err
	})
	var live *unstructured.Unstructured
	var waited time.Duration
	if err == nil {
		live, waited, err = existing(ctx, client, id.Name)
	}
	switch {
	case apierrors.IsNotFound(err):
		resp.Diagnostics.Append(notFound(id, max(waited, time.Second)))
		return
	case err != nil:
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	if namespace != "" && id.Namespace == "" {
		resp.Diagnostics.AddAttributeWarning(path.Root("manifest"), "Namespace ignored", fmt.Sprintf("%s is cluster-scoped: "+
			"the API server ignores metadata.namespace %q. Remove it from the manifest.", id, namespace))
	}
	manager, force := plan.manager()
	skipped := r.engine.skippedFields(plan.manifest(), manifest)
	applied, problem := r.applyFields(ctx, client, live, id, manifest, skipped, manager, force, func(live *unstructured.Unstructured) diag.Diagnostic {
		return taken(live, id, manager)
	})
	if problem != nil {
		resp.Diagnostics.Append(problem)
		return
	}
	state := plan.manifest()
	resp.Diagnostics.Append(recordAppliedState(ctx, &state, id, manifest, manager, skipped, applied, resp.Private)...)
	plan.update(state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, plan.resourceIdentity())...)
	// The state is written: a wait that fails leaves the resource tainted, and
	// replacing it applies the fields again.
	if len(expected) > 0 {
		if _, err := await(ctx, client, id.Name, expected); err != nil {
			resp.Diagnostics.AddError("Fields applied but object not ready", fmt.Sprintf("%s: %s", id, err))
		}
	}
}

func (r *ManifestPatch) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state patchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Terraform before 1.12 sends no identity on refresh, but Framework
	// requires one even when Read removes the resource.
	if resp.Identity != nil && resp.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, state.resourceIdentity())...)
	}
	client, id, err := r.engine.client(state.manifest().identity(), true)
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
	// The patch was of the object that is gone, and went with it. It is
	// planned again, for the object that has the name now.
	if uid := state.UID.ValueString(); uid != "" && string(live.GetUID()) != uid {
		resp.Diagnostics.AddWarning("Object replaced outside Terraform", fmt.Sprintf("%s was deleted and created again outside Terraform: "+
			"its UID is %s, the state has %s. The fields this resource applied went with the old object, so the resource is planned again "+
			"for the new one.", id, live.GetUID(), uid))
		resp.State.RemoveResource(ctx)
		return
	}
	m := state.manifest()
	m.setIdentity(id)
	m.UID = types.StringValue(string(live.GetUID()))
	manager, force := state.manager()
	diags := r.engine.refresh(ctx, &m, id, live, manager, force, req.Private, resp.Private, patchExport)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	state.update(m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, state.resourceIdentity())...)
}

func (r *ManifestPatch) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state patchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// show_drift_values and destroy_behavior change nothing in the cluster.
	pending, _ := req.Private.GetKey(ctx, pendingKey)
	if pending == nil && sameArguments(plan.manifest(), state.manifest()) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}
	timeout, diags := plan.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	manifest, declared, expected, err := prepareApply(ctx, plan.manifest())
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The object stays the one in state. Only its API version follows the manifest.
	id := state.manifest().identity()
	id.APIVersion = declared.APIVersion
	client, id, err := r.engine.client(id, false)
	var live *unstructured.Unstructured
	if err == nil {
		live, err = client.Get(ctx, id.Name, metav1.GetOptions{})
	}
	uid := state.UID.ValueString()
	switch {
	case apierrors.IsNotFound(err):
		resp.Diagnostics.Append(notFound(id, 0))
		return
	case err == nil && uid != "" && string(live.GetUID()) != uid:
		resp.Diagnostics.Append(replaced(id, fmt.Sprintf("the object in the cluster has UID %s, the state has %s", live.GetUID(), uid)))
		return
	case err != nil:
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	manager, force := plan.manager()
	prior, _, _, _ := state.manifest().declared()
	// The fields this resource applied under its previous field manager name
	// move to the new one; other resources' fields stay with the old name.
	// After an import, the name in state is the one the fields were applied as.
	previous := appliedManager(ctx, req.Private)
	if previous == "" {
		previous, _ = state.manager()
	}
	if previous != manager && prior != nil {
		// Checked before the move: fields moved to a name that another
		// resource uses could not be told from that resource's.
		if problem := inUse(live, id, manager, prior, manifest); problem != nil {
			resp.Diagnostics.Append(problem)
			return
		}
		wrote := func(entry metav1.ManagedFieldsEntry) bool {
			return entry.Manager == previous && entry.Subresource == ""
		}
		live, err = guarded(ctx, client, live, func(live *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return takeOver(ctx, client, live, manager, handover{wrote, prior})
		})
		if err != nil {
			resp.Diagnostics.AddError("Cannot move the fields to the new field manager", fmt.Sprintf("%s: %s", id, err))
			return
		}
		// The fields have moved whatever the apply below does, and the state
		// says so: after a failed apply, Terraform knows where they are.
		moved := state
		moved.FieldManager = plan.FieldManager
		resp.Diagnostics.Append(resp.State.Set(ctx, &moved)...)
		resp.Diagnostics.Append(setPrivateString(ctx, resp.Private, managerKey, manager)...)
		resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, moved.resourceIdentity())...)
	}
	skipped := r.engine.skippedFields(plan.manifest(), manifest)
	if keepsNote(ctx, plan.manifest(), req.Private) {
		// lifecycle.ignore_changes kept yaml_body as it was refreshed: what
		// other clients changed stays theirs, as with computed_fields.
		skipped = append(skipped[:len(skipped):len(skipped)], []string{})
	}
	applied, problem := r.applyFields(ctx, client, live, id, manifest, skipped, manager, force, func(live *unstructured.Unstructured) diag.Diagnostic {
		return inUse(live, id, manager, prior, manifest)
	})
	if problem != nil {
		resp.Diagnostics.Append(problem)
		return
	}
	next := plan.manifest()
	resp.Diagnostics.Append(recordAppliedState(ctx, &next, id, manifest, manager, skipped, applied, resp.Private)...)
	plan.update(next)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, plan.resourceIdentity())...)
	if len(expected) > 0 {
		if _, err := await(ctx, client, id.Name, expected); err != nil {
			resp.Diagnostics.AddError("Fields applied but object not ready", fmt.Sprintf("%s: %s", id, err))
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, pendingKey, []byte("true"))...)
		}
	}
}

func (r *ManifestPatch) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state patchModel
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

	manager, _ := state.manager()
	if applied := appliedManager(ctx, req.Private); applied != "" {
		manager = applied
	}
	client, id, err := r.engine.client(state.manifest().identity(), true)
	var live *unstructured.Unstructured
	if err == nil {
		live, err = client.Get(ctx, id.Name, metav1.GetOptions{})
	}
	// Gone with its fields: the object, its kind in every version, or its
	// namespace. Another UID is a new object, which this resource never patched.
	if _, unknownKind := err.(unknownKindError); unknownKind || apierrors.IsNotFound(err) ||
		err == nil && state.UID.ValueString() != "" && string(live.GetUID()) != state.UID.ValueString() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read object", fmt.Sprintf("%s: %s", id, err))
		return
	}
	prior, _, _, _ := state.manifest().declared()
	var foreign []string
	_, err = guarded(ctx, client, live, func(live *unstructured.Unstructured) (*unstructured.Unstructured, error) {
		// Under a name that something else applies with too, nothing tells
		// its fields from those of this resource: all of them stay.
		if foreign = fieldList(foreignFields(live, manager, prior), ""); prior != nil && len(foreign) > 0 {
			return live, nil
		}
		if state.DestroyBehavior.ValueString() != removeFields {
			return relinquish(ctx, client, live, manager)
		}
		// An apply that sets no fields removes those that only this manager
		// applied. The UID keeps it from creating the object again.
		empty := applicable(map[string]any{"apiVersion": live.GetAPIVersion(), "kind": live.GetKind(), "metadata": map[string]any{"name": id.Name}}, id)
		empty.SetUID(live.GetUID())
		empty.SetResourceVersion(live.GetResourceVersion())
		return applyObject(ctx, client, empty, manager, false)
	})
	if prior != nil && len(foreign) > 0 {
		if len(foreign) > 5 {
			foreign = append(foreign[:5], fmt.Sprintf("%d more", len(foreign)-5))
		}
		resp.Diagnostics.AddWarning("Fields not released", fmt.Sprintf("%s: the field manager %q has applied fields that this resource did not set: %s. "+
			"Another resource or client applies to this object under the same name, so the fields of this resource stay with that name and keep their values.",
			id, manager, strings.Join(foreign, ", ")))
		return
	}
	// An object deleted or replaced in the meantime has nothing to release.
	if deleted, other := gone(err); err != nil && !deleted && !other {
		detail := fmt.Sprintf("%s: %s", id, err)
		if apierrors.IsInvalid(err) && state.DestroyBehavior.ValueString() == removeFields {
			detail += "\n\nThe object is not valid without a field this resource applied. To leave the fields in place, " +
				"apply destroy_behavior = \"" + relinquishFields + "\" and destroy again."
		}
		resp.Diagnostics.AddError("Cannot release the fields", detail)
	}
}

// applyFields applies the manifest to the live object. The apply names the
// object's UID, so that it updates the object and never creates it again
// after a deletion. check refuses an apply by what the field manager holds
// on the object, and is asked again for every revision the apply is tried on.
func (r *ManifestPatch) applyFields(ctx context.Context, client dynamic.ResourceInterface, live *unstructured.Unstructured, id identity,
	manifest map[string]any, skipped [][]string, manager string, force bool, check func(*unstructured.Unstructured) diag.Diagnostic) (*unstructured.Unstructured, diag.Diagnostic) {
	var problem diag.Diagnostic
	applied, err := guarded(ctx, client, live, func(live *unstructured.Unstructured) (*unstructured.Unstructured, error) {
		if problem = check(live); problem != nil {
			return nil, nil
		}
		// The apply holds for the object as it was checked: its UID, and its
		// resource version, so that an apply under the same field manager
		// name in between is not overwritten unseen.
		desired := applicable(leavingComputed(manifest, live, manager, skipped), id)
		desired.SetUID(live.GetUID())
		desired.SetResourceVersion(live.GetResourceVersion())
		return r.engine.apply(ctx, client, desired, manager, force)
	})
	deleted, other := gone(err)
	switch {
	case problem != nil:
		return nil, problem
	case err == nil:
		return applied, nil
	case deleted:
		return nil, notFound(id, 0)
	case other:
		return nil, replaced(id, err.Error())
	case stale(err):
		return nil, diag.NewErrorDiagnostic("Cannot apply manifest", fmt.Sprintf("%s: other clients changed the object on every attempt to apply to it, "+
			"for a minute: %s", id, err))
	}
	return nil, applyError(id, err)
}

// taken refuses a new patch under a field manager name that already applies
// fields to the object. Another resource or client uses the name, or this
// resource did and is no longer in the state. An apply would overwrite what
// is there without a conflict, and each resource would release the fields of
// the other. Refusing also keeps a replacement from being created before its
// predecessor is destroyed, whose destroy would then release its fields.
func taken(live *unstructured.Unstructured, id identity, manager string) diag.Diagnostic {
	names := fieldList(managedFields(live, appliedBy(manager)), "")
	if len(names) == 0 {
		return nil
	}
	if len(names) > 5 {
		names = append(names[:5], fmt.Sprintf("%d more", len(names)-5))
	}
	return diag.NewErrorDiagnostic("Field manager in use", fmt.Sprintf("%s: the field manager %q already applies fields to this object: %s. "+
		"For another patch of the object, give this resource a field_manager.name that nothing else uses. "+
		"If this resource applied the fields and is no longer in the state, import it with the ID %q.",
		id, manager, strings.Join(names, ", "), importID(id)+",fieldManager="+manager))
}

// inUse refuses an apply under a field manager name that another resource or
// client applies to the object with: the apply would remove their fields.
func inUse(live *unstructured.Unstructured, id identity, manager string, manifests ...map[string]any) diag.Diagnostic {
	names := fieldList(foreignFields(live, manager, manifests...), "")
	if len(names) == 0 {
		return nil
	}
	if len(names) > 5 {
		names = append(names[:5], fmt.Sprintf("%d more", len(names)-5))
	}
	return diag.NewErrorDiagnostic("Field manager in use", fmt.Sprintf("%s: the field manager %q has applied fields that this manifest "+
		"does not set: %s. Another resource or client applies to this object under the same name, and this apply would remove its fields. "+
		"Give this resource a field_manager.name that nothing else uses.", id, manager, strings.Join(names, ", ")))
}

// foreignFields returns the fields that the manager applied to the object
// and that none of the manifests sets. An apply under that manager removes
// them, so they are those of another resource with the same manager name.
func foreignFields(live *unstructured.Unstructured, manager string, manifests ...map[string]any) map[string]any {
	fields := managedFields(live, appliedBy(manager))
	for _, manifest := range manifests {
		if manifest != nil {
			_, fields = split(fields, manifest)
		}
	}
	return fields
}

// fieldList names the fields of a FieldsV1 set, such as .data.key or
// .spec.containers[name=web].image.
func fieldList(fields map[string]any, prefix string) []string {
	var names []string
	for key, node := range fields {
		kind, text, _ := strings.Cut(key, ":")
		at := prefix
		switch kind {
		case ".":
			continue
		case "f":
			at += "." + text
		case "k":
			var keys map[string]any
			_ = json.Unmarshal([]byte(text), &keys)
			at += "[" + describe(keys) + "]"
		default:
			at += "[" + text + "]"
		}
		members, _ := node.(map[string]any)
		inner := fieldList(members, at)
		if len(inner) == 0 {
			inner = []string{at}
		}
		names = append(names, inner...)
	}
	sort.Strings(names)
	return names
}

// guarded runs a write that names the resource version of the object, and
// runs it again on the object read again while other clients change it in
// between: the status of a DaemonSet in a rollout changes often, and so does
// an object that several patches apply to at once. The object must stay the
// one with the UID live has.
func guarded(ctx context.Context, client dynamic.ResourceInterface, live *unstructured.Unstructured,
	write func(*unstructured.Unstructured) (*unstructured.Unstructured, error)) (*unstructured.Unstructured, error) {
	uid, start := live.GetUID(), time.Now()
	result, err := write(live)
	for attempt := 1; stale(err) && time.Since(start) < time.Minute; attempt++ {
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(time.Duration(min(attempt, 10)) * 20 * time.Millisecond):
		}
		if live, err = client.Get(ctx, live.GetName(), metav1.GetOptions{}); err != nil {
			return nil, err
		}
		if live.GetUID() != uid {
			return nil, fmt.Errorf("%w: its UID is %s, not %s", errReplaced, live.GetUID(), uid)
		}
		result, err = write(live)
	}
	return result, err
}

// stale reports a write the API server refused because the object changed
// since it was read. A field ownership conflict has the same status.
func stale(err error) bool {
	var status apierrors.APIStatus
	if deleted, _ := gone(err); deleted || !apierrors.IsConflict(err) || !errors.As(err, &status) {
		return false
	}
	if details := status.Status().Details; details != nil {
		for _, cause := range details.Causes {
			if cause.Type == metav1.CauseTypeFieldManagerConflict {
				return false
			}
		}
	}
	return true
}

// relinquish gives up the fields the manager applied and keeps their values:
// it removes the manager's entry from the object's field ownership, so that
// any client can apply them without a conflict, as kubectl rewrites the
// entries when it moves an object to server-side apply. The patch names the
// resource version, so a concurrent change makes it fail instead of being
// undone. An entry that is the only one stays: an object without entries
// counts as owned in full by "before-first-apply".
func relinquish(ctx context.Context, client dynamic.ResourceInterface, live *unstructured.Unstructured, manager string) (*unstructured.Unstructured, error) {
	entries := live.GetManagedFields()
	kept := slices.DeleteFunc(slices.Clone(entries), appliedBy(manager))
	if len(kept) == len(entries) || len(kept) == 0 {
		return live, nil
	}
	patch, err := json.Marshal([]map[string]any{
		{"op": "replace", "path": "/metadata/managedFields", "value": kept},
		{"op": "replace", "path": "/metadata/resourceVersion", "value": live.GetResourceVersion()},
	})
	if err != nil {
		return nil, err
	}
	result, err := client.Patch(ctx, live.GetName(), k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
	// The API server keeps the entries it has when it cannot read the new ones.
	if err == nil && len(managedFields(result, appliedBy(manager))) > 0 {
		return nil, fmt.Errorf("the API server kept the entry of field manager %q", manager)
	}
	return result, err
}

// patchExport describes an imported patch: the object and the fields the
// field manager applied to it.
func patchExport(live *unstructured.Unstructured, manager string) map[string]any {
	if len(managedFields(live, appliedBy(manager))) > 0 {
		return exported(live, manager)
	}
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	obj.SetAPIVersion(live.GetAPIVersion())
	obj.SetKind(live.GetKind())
	obj.SetName(live.GetName())
	if live.GetNamespace() != metav1.NamespaceDefault {
		obj.SetNamespace(live.GetNamespace())
	}
	return obj.Object
}
