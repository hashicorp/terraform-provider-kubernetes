// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	policy "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

func (r *PodDisruptionBudgetV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan podDisruptionBudgetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid metadata", "Exactly one metadata block is required.")
		return
	}
	client, _, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	metadata, d := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(d...)
	spec, d := expandPodDisruptionBudgetSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := client.PodDisruptionBudgets(metadata.Namespace).Create(ctx,
		&policy.PodDisruptionBudget{ObjectMeta: metadata, Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating pod disruption budget", err.Error())
		return
	}
	// Preserve the plan's managed values and persist the successful response without
	// a fallible follow-up GET that could orphan a successfully created object.
	setPDBComputedMetadata(&plan, out.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PodDisruptionBudgetV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state podDisruptionBudgetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := podDisruptionBudgetIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid pod disruption budget ID", err.Error())
		return
	}
	client, filters, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := client.PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading pod disruption budget", err.Error())
		return
	}
	metadata, d := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(d...)
	spec, d := flattenPodDisruptionBudgetSpec(ctx, out.Spec, state.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	state.Metadata, state.Spec = metadata, spec
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PodDisruptionBudgetV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state podDisruptionBudgetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 || len(state.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid metadata", "Exactly one metadata block is required in both plan and state.")
		return
	}
	namespace, name, err := podDisruptionBudgetIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid pod disruption budget ID", err.Error())
		return
	}
	client, _, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out *policy.PodDisruptionBudget
	var retryable bool
	summary := "Error updating pod disruption budget"
	err = retry.OnError(retry.DefaultRetry, func(error) bool { return retryable }, func() error {
		retryable = false
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		out, err = client.PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			summary = "Error reading pod disruption budget during update"
			return err
		}
		ops, d := pdbMetadataPatchOps(ctx, out.ObjectMeta, state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() || len(ops) == 0 {
			return nil
		}
		data, err := ops.MarshalJSON()
		if err != nil {
			summary = "Error encoding metadata patch"
			return err
		}
		revision := out.ResourceVersion
		out, err = client.PodDisruptionBudgets(namespace).Patch(ctx, name, k8types.JSONPatchType, data, metav1.PatchOptions{})
		retryable = apierrors.IsConflict(err)
		if pdbIsJSONPatchError(err) && revision != "" {
			// The API server erases JSON Patch test failure details. Only retry its
			// generic 422 when a fresh GET proves our resourceVersion test is stale.
			current, readErr := client.PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
			if readErr != nil {
				return fmt.Errorf("checking resource version after failed patch (%v): %w", err, readErr)
			}
			retryable = current.ResourceVersion != "" && current.ResourceVersion != revision
		}
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(summary, err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	setPDBComputedMetadata(&plan, out.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func pdbIsJSONPatchError(err error) bool {
	var status apierrors.APIStatus
	if !errors.As(err, &status) {
		return false
	}
	value := status.Status()
	return value.Code == http.StatusUnprocessableEntity &&
		value.Reason == metav1.StatusReasonInvalid &&
		value.Message == "the server rejected our request due to an error in our request" &&
		(value.Details == nil || len(value.Details.Causes) == 0)
}

func (r *PodDisruptionBudgetV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state podDisruptionBudgetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := podDisruptionBudgetIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid pod disruption budget ID", err.Error())
		return
	}
	client, _, diags := r.client()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.PodDisruptionBudgets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting pod disruption budget", err.Error())
	}
}

func setPDBComputedMetadata(model *podDisruptionBudgetV1Model, metadata metav1.ObjectMeta) {
	model.ID = types.StringValue(kubernetes.BuildId(metadata))
	model.Metadata[0].Name = types.StringValue(metadata.Name)
	model.Metadata[0].Namespace = types.StringValue(metadata.Namespace)
	model.Metadata[0].UID = types.StringValue(string(metadata.UID))
	model.Metadata[0].Generation = types.Int64Value(metadata.Generation)
	model.Metadata[0].ResourceVersion = types.StringValue(metadata.ResourceVersion)
}

func pdbMetadataPatchOps(ctx context.Context, live metav1.ObjectMeta, state, plan common.MetadataModel) (kubernetes.PatchOperations, diag.Diagnostics) {
	var diags diag.Diagnostics
	merge := func(remote map[string]string, before, after types.Map) (types.Map, types.Map) {
		var oldManaged, newManaged map[string]string
		diags.Append(before.ElementsAs(ctx, &oldManaged, false)...)
		diags.Append(after.ElementsAs(ctx, &newManaged, false)...)
		current := make(map[string]string, len(remote))
		desired := make(map[string]string, len(remote)+len(newManaged))
		for k, v := range remote {
			current[k], desired[k] = v, v
		}
		for k := range oldManaged {
			if _, keep := newManaged[k]; !keep {
				delete(desired, k)
			}
		}
		for k, v := range newManaged {
			desired[k] = v
		}
		oldMap, d := types.MapValueFrom(ctx, types.StringType, current)
		diags.Append(d...)
		newMap, d := types.MapValueFrom(ctx, types.StringType, desired)
		diags.Append(d...)
		return oldMap, newMap
	}
	before, after := common.MetadataModel{}, common.MetadataModel{}
	before.Annotations, after.Annotations = merge(live.Annotations, state.Annotations, plan.Annotations)
	before.Labels, after.Labels = merge(live.Labels, state.Labels, plan.Labels)
	if diags.HasError() {
		return nil, diags
	}
	// Diff the live maps plus managed changes, so adding the first managed key
	// cannot overwrite controller-owned entries. The revision test closes the GET/PATCH race.
	ops := common.MetadataPatchOps("/metadata/", before, after)
	if len(ops) > 0 {
		ops = append(kubernetes.PatchOperations{pdbResourceVersionTest{Value: live.ResourceVersion}}, ops...)
	}
	return ops, diags
}

type pdbResourceVersionTest struct {
	Value string
}

func (pdbResourceVersionTest) GetPath() string { return "/metadata/resourceVersion" }

func (p pdbResourceVersionTest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{Op: "test", Path: p.GetPath(), Value: p.Value})
}
