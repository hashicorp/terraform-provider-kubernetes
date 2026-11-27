// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	// managerKey stores the field manager of the last apply. Only fields that
	// this resource applied under that name move when the name changes.
	managerKey = "manager"
	// pendingKey marks an update whose wait failed. The next plan applies the
	// manifest again and waits again.
	pendingKey = "pending"
	// noteKey holds the drift comment Read added to the end of yaml_body, so
	// that only that text is ever taken off again.
	noteKey = "note"
)

func privateString(ctx context.Context, private privateState, key string) string {
	var text string
	if encoded, _ := private.GetKey(ctx, key); encoded != nil {
		_ = json.Unmarshal(encoded, &text)
	}
	return text
}

func setPrivateString(ctx context.Context, private privateState, key, text string) diag.Diagnostics {
	if text == "" {
		return private.SetKey(ctx, key, nil)
	}
	encoded, _ := json.Marshal(text)
	return private.SetKey(ctx, key, encoded)
}

type privateState interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// appliedManager returns the field manager of the last apply.
func appliedManager(ctx context.Context, private privateState) string {
	var manager string
	if encoded, _ := private.GetKey(ctx, managerKey); encoded != nil {
		_ = json.Unmarshal(encoded, &manager)
	}
	return manager
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

// skippedFields returns the fields that are not compared, as field names:
// computed_fields, and the labels and annotations of the manifest that the
// provider's ignore_labels and ignore_annotations match.
func (r *ManifestV2) skippedFields(m model, manifest map[string]any) [][]string {
	var skipped [][]string
	for _, field := range m.computedFields() {
		if names, err := fieldNames(field); err == nil {
			skipped = append(skipped, names)
		}
	}
	_, filters, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	if diags.HasError() {
		return skipped
	}
	metadata, _ := manifest["metadata"].(map[string]any)
	for field, patterns := range map[string][]string{"annotations": filters.GetIgnoreAnnotations(), "labels": filters.GetIgnoreLabels()} {
		values, _ := metadata[field].(map[string]any)
		for key := range values {
			for _, pattern := range patterns {
				if matched, _ := regexp.MatchString(pattern, key); matched {
					skipped = append(skipped, []string{"metadata", field, key})
					break
				}
			}
		}
	}
	return skipped
}

// recordAppliedState completes the state after an apply. The object attribute
// is read here only when the plan left it open, which it does for a new
// object: ModifyPlan explains why an update keeps it.
func recordAppliedState(ctx context.Context, plan *model, id identity, manifest map[string]any, manager string, skipped [][]string, applied *unstructured.Unstructured, private privateState) diag.Diagnostics {
	plan.setIdentity(id)
	plan.UID = types.StringValue(string(applied.GetUID()))
	plan.Drift = types.ListValueMust(types.StringType, nil)
	_, accepted, _ := liveChanges(manifest, applied, manager, skipped, nil)
	encoded, _ := json.Marshal(accepted)
	diags := private.SetKey(ctx, acceptedKey, encoded)
	diags.Append(setPrivateString(ctx, private, managerKey, manager)...)
	diags.Append(private.SetKey(ctx, pendingKey, nil)...)
	// yaml_body kept as it was refreshed, by lifecycle.ignore_changes, still
	// ends with the drift comment.
	if !keepsNote(ctx, *plan, private) {
		diags.Append(private.SetKey(ctx, noteKey, nil)...)
	}
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
// value open, records it as it was when it was ready.
func waitForReadiness(ctx context.Context, client dynamic.ResourceInterface, id identity, expected []expectation, plan *model, open bool, state *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(expected) == 0 {
		return diags
	}
	live, err := await(ctx, client, id.Name, expected)
	if err != nil {
		diags.AddError("Object applied but not ready", fmt.Sprintf("%s: %s", id, err))
		return diags
	}
	if !open {
		return diags
	}
	diags.Append(observe(ctx, plan, live)...)
	diags.Append(state.Set(ctx, plan)...)
	return diags
}

// uidInvalid reports an apply the API server refused because the object has
// another UID than the one it named.
func uidInvalid(err error) bool {
	var status apierrors.APIStatus
	if !apierrors.IsInvalid(err) || !errors.As(err, &status) || status.Status().Details == nil {
		return false
	}
	for _, cause := range status.Status().Details.Causes {
		if cause.Field == "metadata.uid" {
			return true
		}
	}
	return false
}

func applyError(id identity, err error) diag.Diagnostic {
	detail := err.Error()
	switch {
	// No object name holds a space, so only the API server writes "uid mismatch".
	case strings.Contains(detail, "uid mismatch") || uidInvalid(err):
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

// readBack reads an object that an apply may have created although its
// response was lost. It counts as the one the apply created when it was
// created after the apply was sent and the manager applied fields of it: an
// error such as a failing webhook comes before the API server finds that an
// object exists, and another client may create one in the meantime.
// clockSkew is how far this machine's clock may be ahead of the API server's.
const clockSkew = 5 * time.Second

func readBack(ctx context.Context, client dynamic.ResourceInterface, name, manager string, sent time.Time) *unstructured.Unstructured {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	live, err := client.Get(ctx, name, metav1.GetOptions{})
	// creationTimestamp has whole seconds, from the API server's clock.
	if err != nil || live.GetCreationTimestamp().Time.Before(sent.Add(-clockSkew).Truncate(time.Second)) || len(managedFields(live, appliedBy(manager))) == 0 {
		return nil
	}
	return live
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
	namespace := id.Namespace
	err = retryWhile(ctx, id, func() (err error) {
		client, id, err = r.client(id, false)
		return err
	})
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	if namespace != "" && id.Namespace == "" {
		resp.Diagnostics.AddAttributeWarning(path.Root("manifest"), "Namespace ignored", fmt.Sprintf("%s is cluster-scoped: "+
			"the API server ignores metadata.namespace %q. Remove it from the manifest.", id, namespace))
	}
	manager, force := plan.manager()
	desired := applicable(manifest, id)
	desired.SetResourceVersion(createOnly)
	var applied *unstructured.Unstructured
	var sent time.Time
	err = retryWhile(ctx, id, func() (err error) {
		sent = time.Now()
		applied, err = r.apply(ctx, client, desired, manager, force)
		return err
	})
	if uncertain(err) {
		if found := readBack(ctx, client, id.Name, manager, sent); found != nil {
			resp.Diagnostics.AddWarning("Apply response lost", fmt.Sprintf("%s: %s\n\n"+
				"The object was created by this apply, so the apply took effect.", id, err))
			applied, err = found, nil
		}
	}
	switch {
	case apierrors.IsConflict(err):
		resp.Diagnostics.AddError("Object already exists", fmt.Sprintf("%s already exists in the cluster. "+
			"To manage it with this resource, import it with the ID %q.", id, importID(id)))
		return
	case err != nil:
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	open := plan.Object.IsUnknown()
	resp.Diagnostics.Append(recordAppliedState(ctx, &plan, id, manifest, manager, r.skippedFields(plan, manifest), applied, resp.Private)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, plan.resourceIdentity())...)
	// The state is written: a wait that fails leaves the object tainted.
	resp.Diagnostics.Append(waitForReadiness(ctx, client, id, expected, &plan, open, &resp.State)...)
}

// driftNote returns the comment that Read adds to the end of yaml_body for the
// fields another client changed, with their live values when values is set.
// The plan shows it removed, which restores the fields, and
// lifecycle.ignore_changes on yaml_body leaves both alone.
func driftNote(body string, paths []string, values map[string]any) string {
	if len(paths) == 0 {
		return ""
	}
	note := "# Changed outside Terraform: " + strings.Join(paths, ", ")
	if values != nil {
		note = "# Changed outside Terraform:"
		for _, path := range paths {
			encoded := "(removed)"
			if value, found := values[path]; found {
				encoded = asciiJSON(value)
			}
			note += "\n#   " + path + ": " + encoded
		}
	}
	if strings.HasSuffix(body, "\n") {
		return note + "\n"
	}
	return "\n" + note
}

// asciiJSON encodes a value as JSON with every character outside ASCII
// escaped: YAML reads some of them, such as U+0085, as line breaks, which
// would end the comment.
func asciiJSON(value any) string {
	encoded, _ := json.Marshal(value)
	var out strings.Builder
	for _, r := range string(encoded) {
		switch {
		case r < 0x80:
			out.WriteRune(r)
		case r > 0xFFFF:
			high, low := utf16.EncodeRune(r)
			fmt.Fprintf(&out, "\\u%04x\\u%04x", high, low)
		default:
			fmt.Fprintf(&out, "\\u%04x", r)
		}
	}
	return out.String()
}

// keepsNote reports whether yaml_body still ends with the drift comment that
// the last refresh added: lifecycle.ignore_changes kept it from changing.
func keepsNote(ctx context.Context, m model, private privateState) bool {
	note := privateString(ctx, private, noteKey)
	return note != "" && strings.HasSuffix(m.YAMLBody.ValueString(), note)
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
	// The plan shows this before an apply or a destroy acts on the new object.
	if uid := state.UID.ValueString(); uid != "" && string(live.GetUID()) != uid {
		resp.Diagnostics.AddWarning("Object replaced outside Terraform", fmt.Sprintf("%s was deleted and created again outside Terraform: "+
			"its UID is %s, the state has %s. This resource manages the new object from now on: an apply updates it and a destroy deletes it. "+
			"Remove the resource from the state with terraform state rm if the new object belongs to someone else.", id, live.GetUID(), uid))
	}

	state.setIdentity(id)
	state.UID = types.StringValue(string(live.GetUID()))
	resp.Diagnostics.Append(observe(ctx, &state, live)...)
	manager, force := state.manager()
	diags := r.refresh(ctx, &state, id, live, manager, force, req.Private, resp.Private, exported)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, state.resourceIdentity())...)
}

// refresh brings the manifest arguments in line with the live object: after
// an import export describes the object, and otherwise the arguments show
// what other clients changed since the last apply. private is the private
// state Terraform sent and updated the one returned.
func (r *ManifestV2) refresh(ctx context.Context, state *model, id identity, live *unstructured.Unstructured, manager string, force bool,
	private, updated privateState, export func(*unstructured.Unstructured, string) map[string]any) diag.Diagnostics {
	var diags diag.Diagnostics
	// Until this resource applies under its own name, as after a move, the
	// fields are the previous manager's.
	if previous := appliedManager(ctx, private); previous != "" && len(managedFields(live, appliedBy(manager))) == 0 {
		manager = previous
	}
	// described is set when the manifest in state no longer describes the
	// object: after an import, or when another client changed it.
	var described map[string]any
	var paths []string
	if state.Manifest.IsNull() && state.YAMLBody.IsNull() {
		described = export(live, manager)
	} else {
		// The drift comment is this resource's own text, not part of the manifest.
		declaration := *state
		if !state.YAMLBody.IsNull() {
			declaration.YAMLBody = types.StringValue(strings.TrimSuffix(state.YAMLBody.ValueString(), privateString(ctx, private, noteKey)))
		}
		manifest, _, _, err := declaration.declared()
		if err != nil || manifest == nil {
			diags.AddError("Invalid manifest in state", fmt.Sprint(err))
			return diags
		}
		accepted := map[string]string{}
		if encoded, _ := private.GetKey(ctx, acceptedKey); encoded != nil {
			_ = json.Unmarshal(encoded, &accepted)
		}
		changed, drifted, values := liveChanges(manifest, live, manager, r.skippedFields(*state, manifest), accepted)
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
			diags.AddWarning("Object changed by another field manager", fmt.Sprintf("%s: another client changed %s. "+
				"Applying the manifest will fail with a field ownership conflict unless these fields are removed from it "+
				"or field_manager.force_conflicts is true.", id, strings.Join(conflicts, ", ")))
		}
		// A manifest given as an object shows the live values in the plan.
		// One given as text names the fields in drift and in a comment that
		// the plan shows removed: its values may be sensitive.
		if state.YAMLBody.IsNull() {
			paths = nil
			if len(drifted) > 0 {
				described = changed
			}
		} else {
			if !state.ShowDriftValues.ValueBool() {
				values = nil
			}
			body := declaration.YAMLBody.ValueString()
			note := driftNote(body, paths, values)
			state.YAMLBody = types.StringValue(body + note)
			diags.Append(setPrivateString(ctx, updated, noteKey, note)...)
		}
	}
	drift, listDiags := types.ListValueFrom(ctx, types.StringType, append([]string{}, paths...))
	diags.Append(listDiags...)
	state.Drift = drift
	if described != nil {
		value, err := fromJSON(ctx, described)
		if err != nil {
			diags.AddError("Cannot read object", fmt.Sprintf("%s: %s", id, err))
			return diags
		}
		state.Manifest = types.DynamicValue(value)
	}
	return diags
}

func (r *ManifestV2) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// show_drift_values and deletion_propagation change nothing in the cluster.
	if sameArguments(plan, state) && !plan.Object.IsUnknown() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	// Fields move to this manager from kubectl's client-side apply and other
	// client-side writers the first time it applies, and from the manager
	// this resource applied as before, when the name changes.
	var sources []handover
	if len(managedFields(live, appliedBy(manager))) == 0 {
		sources = append(sources, handover{clientSide, manifest})
	}
	if previous := appliedManager(ctx, req.Private); previous != "" && previous != manager {
		if prior, _, _, err := state.declared(); err == nil && prior != nil {
			// Client-side writes under that name too, from a moved kubectl_manifest.
			wrote := func(entry metav1.ManagedFieldsEntry) bool {
				return entry.Manager == previous && entry.Subresource == ""
			}
			sources = append(sources, handover{wrote, prior})
		}
	}
	// The move names the resource version: a status update in between makes
	// it read the object again, which must still be the one in state.
	moved, err := takeOver(ctx, client, live, manager, sources...)
	for attempt := 0; apierrors.IsConflict(err) && attempt < 5; attempt++ {
		if live, err = client.Get(ctx, id.Name, metav1.GetOptions{}); err == nil {
			if uid != "" && string(live.GetUID()) != uid {
				resp.Diagnostics.Append(applyError(id, fmt.Errorf("uid mismatch: the object in the cluster has UID %s, the state has %s", live.GetUID(), uid)))
				return
			}
			moved, err = takeOver(ctx, client, live, manager, sources...)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot take over the object's fields", fmt.Sprintf("%s: %s", id, err))
		return
	}
	live = moved
	skipped := r.skippedFields(plan, manifest)
	handOver := skipped
	if keepsNote(ctx, plan, req.Private) {
		// lifecycle.ignore_changes kept yaml_body as it was refreshed: what
		// other clients changed stays theirs, as with computed_fields.
		handOver = append(skipped[:len(skipped):len(skipped)], []string{})
	}
	// The apply names the UID too, for a replacement between here and there.
	desired := applicable(leavingComputed(manifest, live, manager, handOver), id)
	if uid != "" {
		desired.SetUID(k8stypes.UID(uid))
	}
	applied, err := r.apply(ctx, client, desired, manager, force)
	if err != nil {
		resp.Diagnostics.Append(applyError(id, err))
		return
	}
	open := plan.Object.IsUnknown()
	// Fields handed over are not compared, so their differences are reported
	// again rather than taken as the result of this apply.
	resp.Diagnostics.Append(recordAppliedState(ctx, &plan, id, manifest, manager, handOver, applied, resp.Private)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, plan.resourceIdentity())...)
	if diags := waitForReadiness(ctx, client, id, expected, &plan, open, &resp.State); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, pendingKey, []byte("true"))...)
	}
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
		// Dependents such as the pods of a Job are deleted too by default, as
		// kubectl deletes them. Naming the UID keeps the delete from removing
		// another object that took the name since the plan.
		propagation := metav1.DeletePropagationBackground
		if policy := state.DeletionPropagation.ValueString(); policy != "" {
			propagation = metav1.DeletionPropagation(policy)
		}
		options := metav1.DeleteOptions{PropagationPolicy: &propagation}
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
	var last error
	err = wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		live, err := client.Get(ctx, id.Name, metav1.GetOptions{})
		switch {
		case apierrors.IsNotFound(err):
			return true, nil
		case err != nil && transient(err):
			last = err
			return false, nil
		case err != nil:
			return false, err
		}
		finalizers = live.GetFinalizers()
		// Another UID is a new object under the same name; ours is gone.
		return string(live.GetUID()) != state.UID.ValueString(), nil
	})
	if err != nil {
		var detail string
		switch {
		case ctx.Err() == context.Canceled:
			detail = fmt.Sprintf("Interrupted while waiting for %s to be removed.", id)
		case ctx.Err() != nil:
			detail = fmt.Sprintf("%s was not removed within %s.", id, timeout)
		default:
			detail = fmt.Sprintf("%s: %s", id, err)
		}
		if last != nil && ctx.Err() != nil {
			detail += fmt.Sprintf(" The last error was: %s.", last)
		}
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
// without status and the bookkeeping the API server and kubectl add, and
// without the data of a Secret, which the plan would show.
func exported(live *unstructured.Unstructured, manager string) map[string]any {
	obj := applicable(live.Object, identity{Namespace: live.GetNamespace()})
	if owned := managedFields(live, appliedBy(manager)); len(owned) > 0 {
		obj.Object, _ = project(obj.Object, owned).(map[string]any)
		obj.SetAPIVersion(live.GetAPIVersion())
		obj.SetKind(live.GetKind())
		obj.SetName(live.GetName())
		// Field ownership does not include the namespace, and a manifest
		// without one means "default".
		if live.GetNamespace() != metav1.NamespaceDefault {
			obj.SetNamespace(live.GetNamespace())
		}
	} else {
		annotations := obj.GetAnnotations()
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		obj.SetAnnotations(annotations)
		// Another client's data stays its own: the apply takes over only the
		// fields the configuration sets.
		if live.GetAPIVersion() == "v1" && live.GetKind() == "Secret" {
			delete(obj.Object, "data")
			delete(obj.Object, "stringData")
		}
	}
	return obj.Object
}
