// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	provkubernetes "github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	// sdkv2SystemDefaultTimeout is SDKv2's own fallback
	// (helper/schema/resource_data.go:552-582). The SDKv2 resource bounded its
	// default-service-account wait with d.Timeout(schema.TimeoutCreate) while
	// declaring only a Delete timeout, so that call returned exactly this. Using
	// it keeps the wait's bound unchanged without adding a `create` key to the
	// published timeouts block.
	sdkv2SystemDefaultTimeout = 20 * time.Minute

	// defaultNamespaceDeleteTimeout matches schema.DefaultTimeout(5*time.Minute)
	// on the SDKv2 resource.
	defaultNamespaceDeleteTimeout = 5 * time.Minute

	namespaceAPIVersion = "v1"
	namespaceKind       = "Namespace"
)

func (r *NamespaceV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NamespaceV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, ignoreAnnotations, ignoreLabels, ok := r.client(&resp.Diagnostics)
	if !ok {
		return
	}

	timeout, d := plan.Timeouts.Create(ctx, sdkv2SystemDefaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The schema validators make an omitted or empty metadata block a plan-time
	// error, so this should be unreachable. Guard anyway: an unguarded
	// plan.Metadata[0] is the exact panic K8S-MIGRATE-002 describes, and a
	// provider panic is far worse than a diagnostic.
	if len(plan.Metadata) == 0 {
		resp.Diagnostics.AddError(
			"Missing metadata block",
			"The namespace has no metadata block. This is a provider bug.",
		)
		return
	}

	meta := plan.Metadata[0]
	objMeta := metav1.ObjectMeta{
		Name:         stringOrEmpty(meta.Name),
		GenerateName: stringOrEmpty(meta.GenerateName),
		Annotations:  expandStringMap(ctx, meta.Annotations, &resp.Diagnostics),
		Labels:       expandStringMap(ctx, meta.Labels, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := corev1.Namespace{ObjectMeta: objMeta}
	out, err := conn.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating Namespace",
			fmt.Sprintf("Failed to create namespace %q: %s", nameForError(objMeta), err.Error()),
		)
		return
	}

	if plan.WaitForDefaultServiceAccount.ValueBool() {
		// Wait on the name the API RETURNED: the configured name is empty when
		// generate_name is used, so waiting on it would query "".
		if err := waitForDefaultServiceAccount(ctx, conn, out.Name); err != nil {
			resp.Diagnostics.AddError(
				"error waiting for the default service account",
				fmt.Sprintf("Failed waiting for the default service account in namespace %q: %s", out.Name, err.Error()),
			)
			return
		}
	}

	// On Create the owned metadata keys and the empty-value conventions both
	// come from the PLAN: it is the configuration, and there is no prior state.
	found, diags := r.refreshModel(ctx, conn, out.Name, &plan, ignoreAnnotations, ignoreLabels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"error reading Namespace after create",
			fmt.Sprintf("Namespace %q disappeared immediately after it was created.", out.Name),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	setNamespaceIdentity(ctx, resp.Identity, out.Name, &resp.Diagnostics)
}

func (r *NamespaceV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, ignoreAnnotations, ignoreLabels, ok := r.client(&resp.Diagnostics)
	if !ok {
		return
	}

	// On refresh both the owned-key allow-list and the empty-value conventions
	// come from prior STATE; there is no plan here.
	found, diags := r.refreshModel(ctx, conn, state.ID.ValueString(), &state, ignoreAnnotations, ignoreLabels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		// An object deleted with kubectl is normal in this ecosystem
		// (rule K8S-CRUD-002).
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	setNamespaceIdentity(ctx, resp.Identity, state.ID.ValueString(), &resp.Diagnostics)
}

func (r *NamespaceV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NamespaceV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, ignoreAnnotations, ignoreLabels, ok := r.client(&resp.Diagnostics)
	if !ok {
		return
	}

	timeout, d := plan.Timeouts.Update(ctx, sdkv2SystemDefaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name := state.ID.ValueString()

	// Reproduce patchMetadata: diff only the keys Terraform manages, so that
	// controller-owned and provider-ignored annotations and labels are never
	// mentioned in the patch and therefore survive (rule K8S-CRUD-006). A whole
	// object Update, or rebuilding metadata from filtered state, would delete
	// them on a change to an unrelated field.
	if len(plan.Metadata) == 0 || len(state.Metadata) == 0 {
		resp.Diagnostics.AddError(
			"Missing metadata block",
			"The namespace has no metadata block. This is a provider bug.",
		)
		return
	}
	oldMeta, newMeta := state.Metadata[0], plan.Metadata[0]
	ops := make(provkubernetes.PatchOperations, 0)
	ops = append(ops, diffMetadataMap(ctx, "/metadata/annotations", oldMeta.Annotations, newMeta.Annotations, &resp.Diagnostics)...)
	ops = append(ops, diffMetadataMap(ctx, "/metadata/labels", oldMeta.Labels, newMeta.Labels, &resp.Diagnostics)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating Namespace",
			fmt.Sprintf("Failed to marshal update operations for namespace %q: %s", name, err.Error()),
		)
		return
	}

	out, err := conn.CoreV1().Namespaces().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating Namespace",
			fmt.Sprintf("Failed to update namespace %q: %s", name, err.Error()),
		)
		return
	}

	found, diags := r.refreshModel(ctx, conn, out.Name, &plan, ignoreAnnotations, ignoreLabels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"error reading Namespace after update",
			fmt.Sprintf("Namespace %q disappeared immediately after it was updated.", out.Name),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	setNamespaceIdentity(ctx, resp.Identity, out.Name, &resp.Diagnostics)
}

func (r *NamespaceV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, _, _, ok := r.client(&resp.Diagnostics)
	if !ok {
		return
	}

	timeout, d := state.Timeouts.Delete(ctx, defaultNamespaceDeleteTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name := state.ID.ValueString()
	err := conn.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		// Namespace deletion, garbage collection and finalizers routinely
		// remove the object before this call lands (rule K8S-CRUD-003).
		if apierrors.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"error deleting Namespace",
			fmt.Sprintf("Failed to delete namespace %q: %s", name, err.Error()),
		)
		return
	}

	// SDKv2 waited for the namespace to leave the Terminating phase with a
	// StateChangeConf bounded by the delete timeout. Namespace deletion is
	// asynchronous and can take minutes while contained objects are reclaimed;
	// returning early would let Terraform recreate the name before the API has
	// released it.
	for {
		out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return
			}
			resp.Diagnostics.AddError(
				"error deleting Namespace",
				fmt.Sprintf("Failed to confirm namespace %q was deleted: %s", name, err.Error()),
			)
			return
		}

		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"error deleting Namespace",
				fmt.Sprintf("Timed out waiting for namespace %q to be deleted; last phase was %q.", name, out.Status.Phase),
			)
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (r *NamespaceV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// SDKv2's resourceIdentityImportNonNamespaced accepted BOTH forms: it used
	// the ID string when one was given and fell back to identity otherwise. The
	// ID form is the documented one, so handling identity alone would break
	// every existing `terraform import kubernetes_namespace_v1.x name`.
	name := req.ID
	if name == "" {
		var identity NamespaceV1IdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		name = identity.Name.ValueString()
	}

	conn, ignoreAnnotations, ignoreLabels, ok := r.client(&resp.Diagnostics)
	if !ok {
		return
	}

	state := NamespaceV1Model{
		// A zero timeouts.Value has a nil object type and fails State.Set, so
		// the null has to carry the block's own attribute types.
		Timeouts: timeouts.Value{
			Object: types.ObjectNull(map[string]attr.Type{
				"delete": types.StringType,
			}),
		},
		// SDKv2 zero-filled this on import: ResourceData.State() wrote d.Get for
		// every top-level field, and the zero of a bool is false. Leaving it
		// null is NOT equivalent — the schema default then plans null -> false,
		// and an `import` block, which requires the post-import plan to be a
		// genuine no-op, fails with
		//   expected a no-op import operation, got ["update"] action
		// ImportStateVerifyIgnore does not cover that path; it only relaxes the
		// legacy ImportStateVerify comparison, so the pre-existing ignore entry
		// hid this in the older tests but the identity import test caught it.
		WaitForDefaultServiceAccount: types.BoolValue(false),
		Metadata:                     []NamespaceMetadataModel{{}},
	}

	found, diags := r.refreshModel(ctx, conn, name, &state, ignoreAnnotations, ignoreLabels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"error importing Namespace",
			fmt.Sprintf("Namespace %q was not found.", name),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	setNamespaceIdentity(ctx, resp.Identity, name, &resp.Diagnostics)
}

// refreshModel reads the namespace and writes its metadata and id into model,
// leaving every other field (timeouts, wait_for_default_service_account)
// untouched. It reports false when the namespace no longer exists.
//
// model carries the prior values on the way in — the plan on Create and Update,
// state on Read — and those serve two distinct purposes: they name the
// annotation and label keys the practitioner owns, and they decide what an
// empty live value is written as.
func (r *NamespaceV1) refreshModel(
	ctx context.Context,
	conn *kubernetes.Clientset,
	name string,
	model *NamespaceV1Model,
	ignoreAnnotations, ignoreLabels []string,
) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, diags
		}
		diags.AddError(
			"error reading Namespace",
			fmt.Sprintf("Failed to read namespace %q: %s", name, err.Error()),
		)
		return false, diags
	}

	var prior NamespaceMetadataModel
	if len(model.Metadata) > 0 {
		prior = model.Metadata[0]
	}

	objMeta := *out.ObjectMeta.DeepCopy()
	provkubernetes.FilterMetadataKeys(
		&objMeta,
		toSDKMap(ctx, prior.Annotations, &diags),
		toSDKMap(ctx, prior.Labels, &diags),
		ignoreAnnotations, ignoreLabels,
	)
	if diags.HasError() {
		return false, diags
	}

	annotations, d := mapOrPrior(ctx, objMeta.Annotations, prior.Annotations)
	diags.Append(d...)
	labels, d := mapOrPrior(ctx, objMeta.Labels, prior.Labels)
	diags.Append(d...)
	if diags.HasError() {
		return false, diags
	}

	model.Metadata = []NamespaceMetadataModel{{
		Annotations:  annotations,
		GenerateName: stringOrPrior(objMeta.GenerateName, prior.GenerateName),
		Generation:   types.Int64Value(objMeta.Generation),
		Labels:       labels,
		// name is Optional+Computed: the API value is authoritative and is
		// never nulled, which is also what makes generate_name round-trip.
		Name:            types.StringValue(objMeta.Name),
		ResourceVersion: types.StringValue(objMeta.ResourceVersion),
		UID:             types.StringValue(string(objMeta.UID)),
	}}
	model.ID = types.StringValue(objMeta.Name)

	return true, diags
}

// waitForDefaultServiceAccount polls until the namespace's default service
// account exists, or ctx (already bounded by the create timeout) expires.
func waitForDefaultServiceAccount(ctx context.Context, conn *kubernetes.Clientset, namespace string) error {
	for {
		_, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, "default", metav1.GetOptions{})
		if err == nil {
			return nil
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for the default service account in namespace %q", namespace)
		case <-time.After(2 * time.Second):
		}
	}
}

func setNamespaceIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, name string, diags *diag.Diagnostics) {
	// Identity is only present when the client negotiated support for it.
	if identity == nil {
		return
	}
	diags.Append(identity.Set(ctx, NamespaceV1IdentityModel{
		APIVersion: types.StringValue(namespaceAPIVersion),
		Kind:       types.StringValue(namespaceKind),
		Name:       types.StringValue(name),
	})...)
}

// ---------------------------------------------------------------------------
// null / empty conventions
// ---------------------------------------------------------------------------

// mapOrPrior decides what to store for an Optional, non-Computed metadata map
// whose filtered live value is empty.
//
// Writing the wrong one is not cosmetic: on Create and Update it is "Provider
// produced inconsistent result after apply", and on refresh it is a diff the
// practitioner cannot resolve. Released 3.2.1 persists null for an omitted map,
// so null is also what keeps an upgraded namespace's plan empty.
//
// So an empty live value keeps the prior KNOWN empty when there is one — the
// planned value on Create and Update, the state value on refresh — and becomes
// null otherwise. That preserves an explicitly configured `annotations = {}`
// while leaving an omitted one null. The one-time state upgrader is the only
// place that nulls empties unconditionally, because upgrading is the one moment
// where the two cases are genuinely indistinguishable.
func mapOrPrior(ctx context.Context, live map[string]string, prior types.Map) (types.Map, diag.Diagnostics) {
	if len(live) > 0 {
		return types.MapValueFrom(ctx, types.StringType, live)
	}
	if isKnownEmptyMap(prior) {
		return prior, nil
	}
	return types.MapNull(types.StringType), nil
}

// stringOrPrior is mapOrPrior for generate_name, whose SDKv2 zero is "".
func stringOrPrior(live string, prior types.String) types.String {
	if live != "" {
		return types.StringValue(live)
	}
	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
		return prior
	}
	return types.StringNull()
}

func isKnownEmptyMap(v types.Map) bool {
	return !v.IsNull() && !v.IsUnknown() && len(v.Elements()) == 0
}

// ---------------------------------------------------------------------------
// conversions
// ---------------------------------------------------------------------------

// stringOrEmpty mirrors what the SDKv2 expander saw. expandMetadata read
// m["name"] straight out of the block, and an unset Optional+Computed field is
// "" there. During Create the Framework marks that same field unknown, so both
// null and unknown have to collapse to "" for the API payload to be identical.
func stringOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

// expandStringMap converts a metadata map for the API payload. It returns nil
// rather than an empty map for an absent value, matching expandMetadata, which
// only assigned the field when len > 0.
func expandStringMap(ctx context.Context, v types.Map, diags *diag.Diagnostics) map[string]string {
	if v.IsNull() || v.IsUnknown() || len(v.Elements()) == 0 {
		return nil
	}
	out := make(map[string]string, len(v.Elements()))
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	return out
}

// toSDKMap converts a metadata map into the map[string]interface{} shape the
// provider's ownership helpers consume.
func toSDKMap(ctx context.Context, v types.Map, diags *diag.Diagnostics) map[string]interface{} {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	typed := make(map[string]string, len(v.Elements()))
	diags.Append(v.ElementsAs(ctx, &typed, false)...)
	out := make(map[string]interface{}, len(typed))
	for k, val := range typed {
		out[k] = val
	}
	return out
}

// diffMetadataMap reproduces one branch of patchMetadata: SDKv2 only called
// diffStringMap when d.HasChange reported a change, so an unchanged map
// contributes no operations at all. Without the equality guard an unchanged
// empty map would emit an add operation rewriting the whole collection.
func diffMetadataMap(ctx context.Context, path string, oldV, newV types.Map, diags *diag.Diagnostics) provkubernetes.PatchOperations {
	oldMap := toSDKMap(ctx, oldV, diags)
	newMap := toSDKMap(ctx, newV, diags)
	if diags.HasError() {
		return nil
	}
	if len(oldMap) == 0 && len(newMap) == 0 {
		return nil
	}
	if reflect.DeepEqual(oldMap, newMap) {
		return nil
	}
	return provkubernetes.DiffStringMap(path, oldMap, newMap)
}

func nameForError(meta metav1.ObjectMeta) string {
	if meta.Name != "" {
		return meta.Name
	}
	return meta.GenerateName + "*"
}
