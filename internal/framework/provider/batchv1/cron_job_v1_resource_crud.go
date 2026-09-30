// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batch "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const cronJobDeleteTimeout = time.Minute

func (r *CronJobV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CronJobV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clients, _, diagnostics := r.sdkv2Meta()
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	metadata, diagnostics := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diagnostics...)
	spec, diagnostics := expandCronJobSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	out, err := conn.BatchV1().CronJobs(metadata.Namespace).Create(ctx, &batch.CronJob{
		ObjectMeta: metadata, Spec: spec,
	}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating CronJob", err.Error())
		return
	}

	// The object exists now. Keep its ID even if response conversion subsequently
	// fails, rather than orphaning it and conflicting on the next apply.
	cronJobAssignedMetadata(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, cronJobIdentity(out.Namespace, out.Name))...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Spec, diagnostics = cronJobAppliedSpec(ctx, plan.Spec, out.Spec)
	resp.Diagnostics.Append(diagnostics...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *CronJobV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CronJobV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := cronJobIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid CronJob ID", err.Error())
		return
	}
	clients, filters, diagnostics := r.sdkv2Meta()
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	out, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading CronJob", err.Error())
		return
	}
	metadata, diagnostics := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diagnostics...)
	spec, diagnostics := flattenCronJobSpec(ctx, out.Spec, state.Spec)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Metadata = metadata
	state.Spec = spec
	state.ID = types.StringValue(out.Namespace + "/" + out.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, cronJobIdentity(out.Namespace, out.Name))...)
	}
}

func (r *CronJobV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CronJobV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := cronJobIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid CronJob ID", err.Error())
		return
	}
	clients, _, diagnostics := r.sdkv2Meta()
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	plannedMeta, diagnostics := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diagnostics...)
	previousMeta, diagnostics := common.ExpandNamespacedMetadata(ctx, state.Metadata)
	resp.Diagnostics.Append(diagnostics...)
	plannedSpec, diagnostics := cronJobSpecForUpdate(ctx, req, plan.Spec, state.Spec)
	resp.Diagnostics.Append(diagnostics...)
	previousSpec, diagnostics := expandCronJobSpec(ctx, state.Spec)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	previousPayload, err := json.Marshal(&batch.CronJob{ObjectMeta: previousMeta, Spec: previousSpec})
	if err != nil {
		resp.Diagnostics.AddError("Invalid prior CronJob payload", err.Error())
		return
	}
	plannedPayload, err := json.Marshal(&batch.CronJob{ObjectMeta: plannedMeta, Spec: plannedSpec})
	if err != nil {
		resp.Diagnostics.AddError("Invalid planned CronJob payload", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	current, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading CronJob before update", err.Error())
		return
	}

	out := current
	// Timeouts and legacy empty-to-null normalization can change Terraform state
	// without changing the Kubernetes payload. Never write those changes remotely.
	if !bytes.Equal(previousPayload, plannedPayload) {
		// A fresh GET retains resourceVersion, generated name, owner references and
		// finalizers. Only previously managed map keys may be removed.
		cronJobMergeMetadata(&current.ObjectMeta, previousMeta, plannedMeta)
		jobMetadata := current.Spec.JobTemplate.ObjectMeta
		cronJobMergeMetadata(&jobMetadata, previousSpec.JobTemplate.ObjectMeta, plannedSpec.JobTemplate.ObjectMeta)
		podMetadata := current.Spec.JobTemplate.Spec.Template.ObjectMeta
		cronJobMergeMetadata(&podMetadata, previousSpec.JobTemplate.Spec.Template.ObjectMeta, plannedSpec.JobTemplate.Spec.Template.ObjectMeta)
		plannedSpec.JobTemplate.ObjectMeta = jobMetadata
		plannedSpec.JobTemplate.Spec.Template.ObjectMeta = podMetadata
		current.Spec = plannedSpec

		out, err = conn.BatchV1().CronJobs(namespace).Update(ctx, current, metav1.UpdateOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating CronJob", err.Error())
			return
		}
	}
	cronJobAssignedMetadata(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, cronJobIdentity(out.Namespace, out.Name))...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Spec, diagnostics = cronJobAppliedSpec(ctx, plan.Spec, out.Spec)
	resp.Diagnostics.Append(diagnostics...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *CronJobV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CronJobV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diagnostics := state.Timeouts.Delete(ctx, cronJobDeleteTimeout)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := cronJobIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid CronJob ID", err.Error())
		return
	}
	clients, _, diagnostics := r.sdkv2Meta()
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err = conn.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting CronJob", err.Error())
		return
	}
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		_, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("CronJob %s/%s still exists", namespace, name))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for CronJob deletion", err.Error())
	}
}

func expandCronJobSpec(ctx context.Context, value types.List) (batch.CronJobSpec, diag.Diagnostics) {
	raw, diagnostics := legacyValue(ctx, value)
	if diagnostics.HasError() {
		return batch.CronJobSpec{}, diagnostics
	}
	items, ok := raw.([]interface{})
	if !ok || len(items) != 1 {
		diagnostics.AddError("Invalid CronJob specification", "Exactly one spec block is required.")
		return batch.CronJobSpec{}, diagnostics
	}
	object, ok := value.Elements()[0].(types.Object)
	if !ok {
		diagnostics.AddError("Invalid CronJob specification", "Expected a spec object.")
		return batch.CronJobSpec{}, diagnostics
	}
	templates, ok := object.Attributes()["job_template"].(types.List)
	if !ok || len(templates.Elements()) != 1 {
		diagnostics.AddError("Invalid CronJob specification", "Exactly one job_template block is required.")
		return batch.CronJobSpec{}, diagnostics
	}
	template, ok := templates.Elements()[0].(types.Object)
	if !ok {
		diagnostics.AddError("Invalid CronJob specification", "Expected a job_template object.")
		return batch.CronJobSpec{}, diagnostics
	}
	jobValue, ok := template.Attributes()["spec"].(types.List)
	if !ok {
		diagnostics.AddError("Invalid CronJob specification", "Expected a job_template spec block.")
		return batch.CronJobSpec{}, diagnostics
	}
	jobSpec, jobDiagnostics := expandJobSpec(ctx, jobValue)
	diagnostics.Append(jobDiagnostics...)
	if diagnostics.HasError() {
		return batch.CronJobSpec{}, diagnostics
	}
	spec, err := kubernetes.ExpandCronJobSpecV1(items)
	if err != nil {
		diagnostics.AddError("Invalid CronJob specification", err.Error())
		return batch.CronJobSpec{}, diagnostics
	}
	// Preserve typed optional-field presence instead of the SDK's null-to-zero conversion.
	spec.JobTemplate.Spec = jobSpec
	return spec, diagnostics
}

func cronJobSpecForUpdate(ctx context.Context, req resource.UpdateRequest, planned, prior types.List) (batch.CronJobSpec, diag.Diagnostics) {
	if req.Config.Schema == nil {
		return expandCronJobSpec(ctx, planned)
	}
	var configured types.List
	diagnostics := req.Config.GetAttribute(ctx, path.Root("spec"), &configured)
	if diagnostics.HasError() {
		return batch.CronJobSpec{}, diagnostics
	}
	plannedRaw, err := planned.ToTerraformValue(ctx)
	if err != nil {
		diagnostics.AddError("Invalid CronJob plan", err.Error())
		return batch.CronJobSpec{}, diagnostics
	}
	priorRaw, err := prior.ToTerraformValue(ctx)
	if err != nil {
		diagnostics.AddError("Invalid CronJob state", err.Error())
		return batch.CronJobSpec{}, diagnostics
	}
	configuredRaw, err := configured.ToTerraformValue(ctx)
	if err != nil {
		diagnostics.AddError("Invalid CronJob configuration", err.Error())
		return batch.CronJobSpec{}, diagnostics
	}
	// Unconfigured computed values may become unknown on an unrelated edit.
	// Retain their prior API values for the update payload, not in the real plan.
	resolved, ok := jobComparisonValue(blockValueField(cronJobSpecBlock()), plannedRaw, priorRaw, configuredRaw)
	if !ok {
		diagnostics.AddError("Unknown CronJob update value", "The desired CronJob specification contains an unresolved configured value.")
		return batch.CronJobSpec{}, diagnostics
	}
	value, err := cronJobSpecBlock().Type().ValueFromTerraform(ctx, resolved)
	if err != nil {
		diagnostics.AddError("Invalid CronJob update value", err.Error())
		return batch.CronJobSpec{}, diagnostics
	}
	return expandCronJobSpec(ctx, value.(types.List))
}

func flattenCronJobSpec(ctx context.Context, spec batch.CronJobSpec, prior types.List) (types.List, diag.Diagnostics) {
	raw, err := kubernetes.FlattenCronJobSpecV1(spec)
	if err != nil {
		var diagnostics diag.Diagnostics
		diagnostics.AddError("Error reading CronJob specification", err.Error())
		return prior, diagnostics
	}
	return valueFromAPI(ctx, cronJobSpecBlock(), raw, prior)
}

func cronJobAppliedSpec(ctx context.Context, plan types.List, spec batch.CronJobSpec) (types.List, diag.Diagnostics) {
	plannedSpec, diagnostics := expandCronJobSpec(ctx, plan)
	if diagnostics.HasError() {
		return plan, diagnostics
	}
	// Admission and other controllers may add template metadata. It remains on
	// the remote object, but is not part of this apply's known configuration.
	spec = *spec.DeepCopy()
	cronJobKeepPlannedMetadata(&spec.JobTemplate.ObjectMeta, plannedSpec.JobTemplate.ObjectMeta)
	cronJobKeepPlannedMetadata(&spec.JobTemplate.Spec.Template.ObjectMeta, plannedSpec.JobTemplate.Spec.Template.ObjectMeta)
	actual, diagnostics := flattenCronJobSpec(ctx, spec, plan)
	if diagnostics.HasError() {
		return plan, diagnostics
	}
	return preservePlannedValue(ctx, cronJobSpecBlock(), plan, actual)
}

func cronJobKeepPlannedMetadata(actual *metav1.ObjectMeta, planned metav1.ObjectMeta) {
	for key := range actual.Annotations {
		if _, configured := planned.Annotations[key]; !configured {
			delete(actual.Annotations, key)
		}
	}
	for key := range actual.Labels {
		if _, configured := planned.Labels[key]; !configured {
			delete(actual.Labels, key)
		}
	}
}

func cronJobAssignedMetadata(model *CronJobV1Model, out *batch.CronJob) {
	model.ID = types.StringValue(out.Namespace + "/" + out.Name)
	if len(model.Metadata) == 0 {
		return
	}
	model.Metadata[0].Name = types.StringValue(out.Name)
	model.Metadata[0].Namespace = types.StringValue(out.Namespace)
	model.Metadata[0].Generation = types.Int64Value(out.Generation)
	model.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	model.Metadata[0].UID = types.StringValue(string(out.UID))
}

func cronJobMergeMetadata(current *metav1.ObjectMeta, previous, planned metav1.ObjectMeta) {
	current.Annotations = cronJobMergeMap(current.Annotations, previous.Annotations, planned.Annotations)
	current.Labels = cronJobMergeMap(current.Labels, previous.Labels, planned.Labels)
}

func cronJobMergeMap(current, previous, planned map[string]string) map[string]string {
	merged := make(map[string]string, len(current)+len(planned))
	for key, value := range current {
		merged[key] = value
	}
	for key := range previous {
		if _, managed := planned[key]; !managed {
			delete(merged, key)
		}
	}
	for key, value := range planned {
		merged[key] = value
	}
	return merged
}
