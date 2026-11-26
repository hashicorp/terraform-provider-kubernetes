// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// defaultTimeout bounds each operation unless the timeouts block sets one. The
// data source, which has no timeouts block, always uses it.
const defaultTimeout = 10 * time.Minute

func identity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{Name: types.StringValue(name), Kind: types.StringValue("GatewayClass"), APIVersion: types.StringValue(apiVersion)}
}

// afterWrite copies the server-assigned values from a Create or Update response into the plan before it is saved
func (m *gatewayClassModel) afterWrite(ctx context.Context, object *gatewayv1.GatewayClass) diag.Diagnostics {
	m.ID = types.StringValue(object.Name)
	m.Metadata[0].Name = types.StringValue(object.Name)
	m.Metadata[0].UID = types.StringValue(string(object.UID))
	m.Metadata[0].ResourceVersion = types.StringValue(object.ResourceVersion)
	m.Metadata[0].Generation = types.Int64Value(object.Generation)
	status, diags := flattenStatus(ctx, object.Status)
	m.Status = status
	return diags
}

// Create implements [resource.Resource]. It returns once the API server has
// stored the object: acceptance depends on a controller that may not be
// running yet, so status is saved as returned and refreshed by Read. A
// NotFound here means the API is not served, since a cluster-scoped create has
// no namespace to miss; discovery confirms it and the error says to install
// the CRDs.
func (r *GatewayClassResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gatewayClassModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, discover, _, diags := r.clients()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	meta, diags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	object, err := client.Create(ctx, &gatewayv1.GatewayClass{ObjectMeta: meta, Spec: plan.Spec[0].expand()}, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if endpointErr := gatewayClassEndpoint(ctx, discover); endpointErr != nil {
				resp.Diagnostics.AddError("GatewayClass API unavailable", endpointErr.Error())
				return
			}
		}
		resp.Diagnostics.AddError("Error creating GatewayClass", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.afterWrite(ctx, object)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity(object.Name))...)
}

// Read implements [resource.Resource]. A NotFound removes the resource from
// state only after gatewayClassEndpoint confirms the API is still served; if
// the CRDs are gone, or discovery fails, state is kept and Read reports an
// error. Labels and annotations go through common.FlattenMetadata and, except
// on import, are then limited to keys already in state (see managedMetadataMap).
func (r *GatewayClassResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gatewayClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := state.Timeouts.Read(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, discover, filters, diags := r.clients()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	object, err := client.Get(ctx, state.ID.ValueString(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if endpointErr := gatewayClassEndpoint(ctx, discover); endpointErr != nil {
			resp.Diagnostics.AddError("GatewayClass API unavailable", endpointErr.Error())
			return
		}
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading GatewayClass", err.Error())
		return
	}
	prior := state.Metadata
	state.Metadata, diags = common.FlattenMetadata(ctx, object.ObjectMeta, prior, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	// Import adopts observable metadata; refresh must not adopt keys added by controllers.
	if len(prior) == 1 {
		state.Metadata[0].Labels, diags = managedMetadataMap(state.Metadata[0].Labels, prior[0].Labels)
		resp.Diagnostics.Append(diags...)
		state.Metadata[0].Annotations, diags = managedMetadataMap(state.Metadata[0].Annotations, prior[0].Annotations)
		resp.Diagnostics.Append(diags...)
	}
	state.Spec = []gatewayClassSpec{flattenSpec(object.Spec)}
	state.Status, diags = flattenStatus(ctx, object.Status)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity(object.Name))...)
}

// metadataChanges builds the labels or annotations object of a JSON merge
// patch: planned keys are set, and keys in state that are no longer planned
// become null, which deletes them. Keys absent from state, such as those
// added by controllers, are left out, so the server keeps them.
func metadataChanges(old, desired types.Map) map[string]any {
	changes := make(map[string]any)
	desiredElements := desired.Elements()
	for key := range old.Elements() {
		if _, exists := desiredElements[key]; !exists {
			changes[key] = nil
		}
	}
	for key, value := range common.ExpandMapForPatch(desired) {
		changes[key] = value
	}
	return changes
}

// managedMetadataMap narrows observed labels or annotations to the keys in
// prior state. common.FlattenMetadata drops internal and ignored keys but keeps
// any other key a controller adds, and Terraform would then plan to delete it
// on every apply. Managed keys keep their observed values, so out-of-band edits
// and removals still show as drift; a null prior map stays null.
func managedMetadataMap(observed, prior types.Map) (types.Map, diag.Diagnostics) {
	if prior.IsNull() {
		return types.MapNull(types.StringType), nil
	}
	values := observed.Elements()
	managed := prior.Elements()
	for key := range values {
		if _, exists := managed[key]; !exists {
			delete(values, key)
		}
	}
	return types.MapValue(types.StringType, values)
}

// Update implements [resource.Resource] as one JSON merge patch. Sending the
// resource_version from state makes the API reject the patch with a conflict
// if the object changed since Terraform last read or wrote it, instead of
// overwriting that change. Labels and annotations are patched key by key (see
// metadataChanges); the spec is sent whole.
func (r *GatewayClassResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state gatewayClassModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, _, _, diags := r.clients()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	spec := plan.Spec[0].expand()
	// The API types use omitempty, so marshalling them would drop a removed
	// description, parametersRef or parametersRef.namespace instead of sending
	// the null that makes a merge patch delete it. Build the spec by hand.
	var parameters any
	if spec.ParametersRef != nil {
		ref := spec.ParametersRef
		parameters = map[string]any{"group": ref.Group, "kind": ref.Kind, "name": ref.Name, "namespace": ref.Namespace}
	}
	patch := map[string]any{
		"metadata": map[string]any{
			"resourceVersion": state.Metadata[0].ResourceVersion.ValueString(),
			"labels":          metadataChanges(state.Metadata[0].Labels, plan.Metadata[0].Labels),
			"annotations":     metadataChanges(state.Metadata[0].Annotations, plan.Metadata[0].Annotations),
		},
		"spec": map[string]any{"controllerName": spec.ControllerName, "description": spec.Description, "parametersRef": parameters},
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		resp.Diagnostics.AddError("Error encoding GatewayClass update", err.Error())
		return
	}
	object, err := client.Patch(ctx, state.ID.ValueString(), ktypes.MergePatchType, payload, metav1.PatchOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error updating GatewayClass", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.afterWrite(ctx, object)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity(object.Name))...)
}

// Delete implements [resource.Resource]. Controllers are expected to hold
// gatewayv1.GatewayClassFinalizerGatewaysExist while Gateways use the class,
// so Delete polls every second until the object is gone. On timeout it reports
// the remaining finalizers without removing them, which would bypass that
// protection. A NotFound, from the request or while polling, counts as deleted
// only once discovery confirms the API is still served.
func (r *GatewayClassResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gatewayClassModel
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
	client, discover, _, diags := r.clients()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := client.Delete(ctx, state.ID.ValueString(), metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		if endpointErr := gatewayClassEndpoint(ctx, discover); endpointErr != nil {
			resp.Diagnostics.AddError("GatewayClass API unavailable", endpointErr.Error())
		}
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting GatewayClass", err.Error())
		return
	}
	var finalizers []string
	err = wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		object, err := client.Get(ctx, state.ID.ValueString(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			if endpointErr := gatewayClassEndpoint(ctx, discover); endpointErr != nil {
				return false, endpointErr
			}
			return true, nil
		}
		if err != nil {
			return false, err
		}
		finalizers = object.Finalizers
		return false, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for GatewayClass deletion", fmt.Sprintf("%s. Remaining finalizers: %v. Remove dependent Gateways and allow the controller to release finalizers.", err, finalizers))
	}
}
