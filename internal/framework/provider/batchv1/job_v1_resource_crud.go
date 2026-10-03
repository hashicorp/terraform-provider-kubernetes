// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batchapi "k8s.io/api/batch/v1"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/batch/v1"
)

const jobDefaultTimeout = time.Minute

func (r *JobV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JobV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	timeout, diags := plan.Timeouts.Create(ctx, jobDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
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
	spec, diags := expandJobSpec(ctx, plan.Spec, true, &req.Config, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := conn.BatchV1().Jobs(metadata.Namespace).Create(ctx, &batchapi.Job{ObjectMeta: metadata, Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Job", err.Error())
		return
	}
	// The Job exists now: keep it in state even if waiting for it fails.
	resp.Diagnostics.Append(jobWriteResult(ctx, &resp.State, req.Plan, plan, out, filters)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, podTemplateMetadataOwnershipInitialized, []byte("true"))...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, jobIdentity(out.Namespace, out.Name))...)
	}
	if resp.Diagnostics.HasError() || !plan.WaitForCompletion.ValueBool() {
		return
	}
	if err := waitForJobCompletion(ctx, conn.BatchV1().Jobs(out.Namespace), out.Namespace, out.Name, timeout); err != nil {
		resp.Diagnostics.AddError("Error waiting for Job completion", err.Error())
	}
}

func (r *JobV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JobV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Job ID", err.Error())
		return
	}
	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	out, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Job", err.Error())
		return
	}
	resp.Diagnostics.Append(flattenJob(ctx, out, &state, filters, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, jobIdentity(namespace, name))...)
	}
}

func (r *JobV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state JobV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	timeout, diags := plan.Timeouts.Update(ctx, jobDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Job ID", err.Error())
		return
	}
	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid Job metadata", "Expected exactly one metadata block in the state and plan.")
		return
	}
	ops, err := jobMetadataPatchOps(ctx, conn.BatchV1().Jobs(namespace), name,
		state.Metadata[0].MetadataBase, plan.Metadata[0].MetadataBase)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Job metadata for update", err.Error())
		return
	}
	specOps, diags := patchJobSpec(ctx, state.Spec, plan.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops = append(ops, specOps...)
	var out *batchapi.Job
	if len(ops) == 0 {
		out, err = conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		data, marshalErr := ops.MarshalJSON()
		if marshalErr != nil {
			resp.Diagnostics.AddError("Failed to marshal Job update", marshalErr.Error())
			return
		}
		out, err = conn.BatchV1().Jobs(namespace).Patch(ctx, name, k8stypes.JSONPatchType, data, metav1.PatchOptions{})
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Job", err.Error())
		return
	}
	resp.Diagnostics.Append(jobWriteResult(ctx, &resp.State, req.Plan, plan, out, filters)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, podTemplateMetadataOwnershipInitialized, []byte("true"))...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, jobIdentity(namespace, name))...)
	}
	if resp.Diagnostics.HasError() || !plan.WaitForCompletion.ValueBool() {
		return
	}
	if err := waitForJobCompletion(ctx, conn.BatchV1().Jobs(namespace), namespace, name, timeout); err != nil {
		resp.Diagnostics.AddError("Error waiting for Job completion", err.Error())
	}
}

func (r *JobV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state JobV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	timeout, diags := state.Timeouts.Delete(ctx, jobDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Job ID", err.Error())
		return
	}
	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	propagation := metav1.DeletePropagationForeground
	jobs := conn.BatchV1().Jobs(namespace)
	err = jobs.Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &propagation})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Job", err.Error())
		return
	}
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		_, err := jobs.Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("Job %s/%s still exists", namespace, name))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Job deletion", err.Error())
	}
}

func waitForJobCompletion(ctx context.Context, jobs v1.JobInterface, namespace, name string, timeout time.Duration) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		job, err := jobs.Get(ctx, name, metav1.GetOptions{})
		// Jobs with a zero TTL can disappear before the completion condition is observed.
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		for _, condition := range job.Status.Conditions {
			if condition.Status != coreapi.ConditionTrue {
				continue
			}
			switch condition.Type {
			case batchapi.JobComplete:
				return nil
			case batchapi.JobFailed:
				return retry.NonRetryableError(fmt.Errorf("job: %s/%s is in failed state", namespace, name))
			}
		}
		return retry.RetryableError(fmt.Errorf("job: %s/%s is not in complete state", namespace, name))
	})
}

func flattenJob(ctx context.Context, job *batchapi.Job, model *JobV1Model, filters kubernetes.MetadataFilters, refresh bool) diag.Diagnostics {
	job = job.DeepCopy()
	if job.Spec.ManualSelector == nil || !*job.Spec.ManualSelector {
		removeJobGeneratedLabels(job.Labels)
	}
	metadata, diags := common.FlattenNamespacedMetadata(ctx, job.ObjectMeta, model.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	spec, specDiags := flattenJobSpec(ctx, job.Spec, model.Spec, true, refresh, path.Root("spec"))
	diags.Append(specDiags...)
	if diags.HasError() {
		return diags
	}
	model.Metadata = metadata
	model.Spec = spec
	model.ID = types.StringValue(kubernetes.BuildId(job.ObjectMeta))
	if model.WaitForCompletion.IsNull() || model.WaitForCompletion.IsUnknown() {
		model.WaitForCompletion = types.BoolValue(true)
	}
	return diags
}

// jobWriteResult records the plan after a write, with the values Kubernetes
// chose for those it left unknown.
func jobWriteResult(ctx context.Context, state *tfsdk.State, plan tfsdk.Plan, model JobV1Model, out *batchapi.Job, filters kubernetes.MetadataFilters) diag.Diagnostics {
	return common.SetWriteResult(ctx, state, plan, func(actual *tfsdk.State) diag.Diagnostics {
		model.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
		diags := flattenJob(ctx, out, &model, filters, false)
		return append(diags, actual.Set(ctx, &model)...)
	})
}

func removeJobGeneratedLabels(labels map[string]string) {
	for _, key := range jobGeneratedLabels {
		delete(labels, key)
	}
}

func jobIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue("batch/v1"),
			Kind:       types.StringValue("Job"),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

func patchJobSpec(ctx context.Context, state, plan types.List) (kubernetes.PatchOperations, diag.Diagnostics) {
	var diags diag.Diagnostics
	if state.Equal(plan) {
		return nil, diags
	}
	if len(state.Elements()) != 1 || len(plan.Elements()) != 1 {
		diags.AddError("Invalid Job specification", "Expected exactly one spec block in the state and plan.")
		return nil, diags
	}
	oldObject, oldOK := state.Elements()[0].(types.Object)
	newObject, newOK := plan.Elements()[0].(types.Object)
	if !oldOK || !newOK {
		diags.AddError("Invalid Job specification", "Expected spec to contain an object.")
		return nil, diags
	}
	oldAttrs, newAttrs := oldObject.Attributes(), newObject.Attributes()
	payloadPlan := plan
	if newAttrs["completion_mode"].IsUnknown() {
		// Completion mode is immutable. Its computed unknown must not erase
		// Indexed-only mutable fields while constructing an update payload.
		fields := newObject.Attributes()
		fields["completion_mode"] = oldAttrs["completion_mode"]
		payloadObject, objectDiags := types.ObjectValue(newObject.AttributeTypes(ctx), fields)
		diags.Append(objectDiags...)
		var payloadDiags diag.Diagnostics
		payloadPlan, payloadDiags = types.ListValue(plan.ElementType(ctx), []attr.Value{payloadObject})
		diags.Append(payloadDiags...)
		if diags.HasError() {
			return nil, diags
		}
	}
	spec, expandDiags := expandJobSpec(ctx, payloadPlan, true, nil, path.Root("spec"))
	diags.Append(expandDiags...)
	previousSpec, previousDiags := expandJobSpec(ctx, state, true, nil, path.Root("spec"))
	diags.Append(previousDiags...)
	if diags.HasError() {
		return nil, diags
	}
	fields := []struct {
		name     string
		apiName  string
		value    any
		previous any
	}{
		{"active_deadline_seconds", "activeDeadlineSeconds", spec.ActiveDeadlineSeconds, previousSpec.ActiveDeadlineSeconds},
		{"backoff_limit", "backoffLimit", spec.BackoffLimit, previousSpec.BackoffLimit},
		{"manual_selector", "manualSelector", spec.ManualSelector, previousSpec.ManualSelector},
		{"max_failed_indexes", "maxFailedIndexes", spec.MaxFailedIndexes, previousSpec.MaxFailedIndexes},
		{"parallelism", "parallelism", spec.Parallelism, previousSpec.Parallelism},
		{"ttl_seconds_after_finished", "ttlSecondsAfterFinished", spec.TTLSecondsAfterFinished, previousSpec.TTLSecondsAfterFinished},
	}
	ops := make(kubernetes.PatchOperations, 0, len(fields))
	for _, field := range fields {
		// State written before a field existed holds null where the plan holds its zero default.
		if oldAttrs[field.name].Equal(newAttrs[field.name]) || (oldAttrs[field.name].IsNull() && isZeroAttr(newAttrs[field.name])) {
			continue
		}
		// JSON Patch "add" replaces existing members and also works for an omitted
		// optional field. A null value clears an optional Kubernetes pointer field.
		value := field.value
		if newAttrs[field.name].IsNull() {
			value = nil
		}
		previous := field.previous
		if oldAttrs[field.name].IsNull() {
			previous = nil
		}
		oldJSON, oldErr := json.Marshal(previous)
		newJSON, newErr := json.Marshal(value)
		if oldErr == nil && newErr == nil && bytes.Equal(oldJSON, newJSON) {
			continue
		}
		ops = append(ops, &kubernetes.AddOperation{Path: "/spec/" + field.apiName, Value: value})
	}
	return ops, diags
}

func jobMetadataPatchOps(ctx context.Context, jobs v1.JobInterface, name string, state, plan common.MetadataBase) (kubernetes.PatchOperations, error) {
	annotationsChanged := !plan.Annotations.IsUnknown() && !plan.Annotations.Equal(state.Annotations)
	labelsChanged := !plan.Labels.IsUnknown() && !plan.Labels.Equal(state.Labels)
	if !annotationsChanged && !labelsChanged {
		return nil, nil
	}
	live, err := jobs.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ops := make(kubernetes.PatchOperations, 0)
	for _, field := range []struct {
		name           string
		changed        bool
		current        map[string]string
		prior, planned types.Map
	}{
		{"annotations", annotationsChanged, live.Annotations, state.Annotations, plan.Annotations},
		{"labels", labelsChanged, live.Labels, state.Labels, plan.Labels},
	} {
		if !field.changed {
			continue
		}
		current := make(map[string]interface{}, len(field.current))
		desired := make(map[string]interface{}, len(field.current))
		for key, value := range field.current {
			current[key], desired[key] = value, value
		}
		prior := common.ExpandMapForPatch(field.prior)
		planned := common.ExpandMapForPatch(field.planned)
		for key := range prior {
			if _, retained := planned[key]; !retained {
				delete(desired, key)
			}
		}
		for key, value := range planned {
			desired[key] = value
		}
		if len(current) == 0 && len(desired) == 0 {
			continue
		}
		ops = append(ops, kubernetes.DiffStringMap("/metadata/"+field.name, current, desired)...)
	}
	return ops, nil
}

func isZeroAttr(value attr.Value) bool {
	switch v := value.(type) {
	case types.Int64:
		return v.Equal(types.Int64Value(0))
	case types.Bool:
		return v.Equal(types.BoolValue(false))
	case types.String:
		return v.Equal(types.StringValue(""))
	}
	return false
}
