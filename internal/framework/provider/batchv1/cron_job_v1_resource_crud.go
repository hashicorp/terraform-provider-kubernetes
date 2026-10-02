// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batch "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	k8sretry "k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
)

const cronJobDeleteTimeout = time.Minute

func (r *CronJobV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CronJobV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
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
	metadata, diagnostics := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diagnostics...)
	spec, diagnostics := expandCronJobSpec(ctx, plan.Spec, path.Root("spec"))
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
	resp.Diagnostics.Append(cronJobWriteResult(ctx, &resp.State, plan, out, filters)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, cronJobIdentity(out.Namespace, out.Name))...)
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
	resp.Diagnostics.Append(flattenCronJob(ctx, out, &state, filters)...)
	if resp.Diagnostics.HasError() {
		return
	}
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
	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid CronJob metadata", "Expected exactly one metadata block in the state and plan.")
		return
	}
	previousSpec, diagnostics := expandCronJobSpec(ctx, state.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diagnostics...)
	desiredSpec, diagnostics := cronJobDesiredSpec(ctx, req)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	dynamicClient, err := clients.DynamicClient()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	var out *batch.CronJob
	// Only what the update changes is written: state-only changes, such as
	// timeouts, send nothing, and fields the provider does not manage keep
	// their live values.
	err = k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		raw, err := dynamicClient.Resource(batch.SchemeGroupVersion.WithResource("cronjobs")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		out = &batch.CronJob{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, out); err != nil {
			return err
		}
		ops := common.MetadataPatchOpsAgainstLive("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel, out.ObjectMeta)
		specOps, err := cronJobSpecOps(raw, previousSpec, desiredSpec)
		if err != nil {
			return err
		}
		ops = append(ops, specOps...)
		if len(ops) == 0 {
			return nil
		}
		ops = append(kubernetes.PatchOperations{common.ResourceVersionGuard(raw.GetResourceVersion())}, ops...)
		data, err := ops.MarshalJSON()
		if err != nil {
			return err
		}
		out, err = conn.BatchV1().CronJobs(namespace).Patch(ctx, name, k8stypes.JSONPatchType, data, metav1.PatchOptions{})
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating CronJob", err.Error())
		return
	}
	resp.Diagnostics.Append(cronJobWriteResult(ctx, &resp.State, plan, out, filters)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, cronJobIdentity(out.Namespace, out.Name))...)
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

// expandCronJobSpec converts the single "spec" element with SDKv2's field
// semantics: history limits are sent only when they differ from the defaults.
func expandCronJobSpec(ctx context.Context, value types.List, at path.Path) (batch.CronJobSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	spec, ok := singleObject(value)
	if !ok {
		diags.AddAttributeError(at, "Invalid CronJob specification", "Exactly one known spec block is required.")
		return batch.CronJobSpec{}, diags
	}
	a := spec.Attributes()
	var out batch.CronJobSpec
	if s, _ := knownString(a["concurrency_policy"]); s != "" {
		out.ConcurrencyPolicy = batch.ConcurrencyPolicy(s)
	}
	if n, ok := knownInt64(a["failed_jobs_history_limit"]); ok && n != 1 {
		out.FailedJobsHistoryLimit = ptr.To(int32(n))
	}
	out.Schedule, _ = knownString(a["schedule"])
	if s, _ := knownString(a["timezone"]); s != "" {
		out.TimeZone = ptr.To(s)
	}
	if n, ok := knownInt64(a["starting_deadline_seconds"]); ok && n > 0 {
		out.StartingDeadlineSeconds = ptr.To(n)
	}
	if n, ok := knownInt64(a["successful_jobs_history_limit"]); ok && n != 3 {
		out.SuccessfulJobsHistoryLimit = ptr.To(int32(n))
	}
	if b, ok := a["suspend"].(types.Bool); ok && !b.IsNull() && !b.IsUnknown() {
		out.Suspend = ptr.To(b.ValueBool())
	}
	template, ok := singleObject(a["job_template"])
	if !ok {
		diags.AddAttributeError(at.AtListIndex(0).AtName("job_template"), "Invalid CronJob specification", "Exactly one known job_template block is required.")
		return out, diags
	}
	jobSpec, d := expandJobSpec(ctx, template.Attributes()["spec"].(types.List), false, at.AtListIndex(0).AtName("job_template").AtListIndex(0).AtName("spec"))
	diags.Append(d...)
	out.JobTemplate = batch.JobTemplateSpec{ObjectMeta: expandTemplateMetadata(template.Attributes()["metadata"]), Spec: jobSpec}
	return out, diags
}

func flattenCronJobSpec(ctx context.Context, in batch.CronJobSpec, prior types.List, at path.Path) (types.List, diag.Diagnostics) {
	typ := cronJobSpecBlockType()
	previous := priorAttributes(prior)
	templateType := typ.AttrTypes["job_template"].(types.ListType).ElemType.(types.ObjectType)
	previousTemplate := priorAttributes(previous["job_template"])
	jobSpecPrior, ok := previousTemplate["spec"].(types.List)
	if !ok {
		jobSpecPrior = types.ListNull(jobTemplateSpecType())
	}
	var diags diag.Diagnostics
	jobSpec, d := flattenJobSpec(ctx, in.JobTemplate.Spec, jobSpecPrior, false, at.AtListIndex(0).AtName("job_template").AtListIndex(0).AtName("spec"))
	diags.Append(d...)
	metadata := flattenTemplateMetadata(in.JobTemplate.ObjectMeta, previousTemplate["metadata"], templateType.AttrTypes["metadata"].(types.ListType), nil, &diags)
	if diags.HasError() {
		return types.ListNull(typ), diags
	}
	template := singletonList(templateType, map[string]attr.Value{"metadata": metadata, "spec": jobSpec}, &diags)
	timezone := ""
	if in.TimeZone != nil {
		timezone = *in.TimeZone
	}
	return singletonList(typ, map[string]attr.Value{
		"concurrency_policy":            types.StringValue(string(in.ConcurrencyPolicy)),
		"failed_jobs_history_limit":     types.Int64Value(int64(ptr.Deref(in.FailedJobsHistoryLimit, 0))),
		"job_template":                  template,
		"schedule":                      types.StringValue(in.Schedule),
		"starting_deadline_seconds":     types.Int64Value(ptr.Deref(in.StartingDeadlineSeconds, 0)),
		"successful_jobs_history_limit": types.Int64Value(int64(ptr.Deref(in.SuccessfulJobsHistoryLimit, 0))),
		"suspend":                       types.BoolValue(ptr.Deref(in.Suspend, false)),
		"timezone":                      types.StringValue(timezone),
	}, &diags), diags
}

func flattenCronJob(ctx context.Context, out *batch.CronJob, model *CronJobV1Model, filters kubernetes.MetadataFilters) diag.Diagnostics {
	metadata, diags := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, model.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	spec, d := flattenCronJobSpec(ctx, out.Spec, model.Spec, path.Root("spec"))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	model.Metadata = metadata
	model.Spec = spec
	model.ID = types.StringValue(out.Namespace + "/" + out.Name)
	return diags
}

// cronJobWriteResult records the plan after a write, with the values Kubernetes
// chose for those it left unknown.
func cronJobWriteResult(ctx context.Context, state *tfsdk.State, plan CronJobV1Model, out *batch.CronJob, filters kubernetes.MetadataFilters) diag.Diagnostics {
	plan.ID = types.StringValue(out.Namespace + "/" + out.Name)
	planned := tfsdk.State{Schema: state.Schema}
	diags := planned.Set(ctx, &plan)
	actual := tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)}
	model := plan
	if flattenDiags := flattenCronJob(ctx, out, &model, filters); flattenDiags.HasError() {
		diags.Append(flattenDiags...)
	} else {
		diags.Append(actual.Set(ctx, &model)...)
	}
	merged, err := knownOrActual(planned.Raw, actual.Raw)
	if err != nil {
		diags.AddError("Unable to record CronJob state", err.Error())
		return diags
	}
	state.Raw = merged
	return diags
}

// cronJobDesiredSpec is the planned spec, with the values the plan leaves to
// Kubernetes (unknown, or an API-defaulted string configured as "") kept as
// they were.
func cronJobDesiredSpec(ctx context.Context, req resource.UpdateRequest) (batch.CronJobSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	at := tftypes.NewAttributePath().WithAttributeName("spec")
	config, configOK := valueAt(req.Config.Raw, at)
	plan, planOK := valueAt(req.Plan.Raw, at)
	state, stateOK := valueAt(req.State.Raw, at)
	if !configOK || !planOK || !stateOK {
		diags.AddError("Unable to read CronJob specification", "The spec block is missing from the configuration, plan or state.")
		return batch.CronJobSpec{}, diags
	}
	if resolved, ok := resolveUnconfigured(at, config, plan, state, apiDefaultedStrings(ctx, req.Plan.Schema)); ok {
		plan = resolved
	}
	value, err := types.ListType{ElemType: cronJobSpecBlockType()}.ValueFromTerraform(ctx, plan)
	if err != nil {
		diags.AddError("Unable to read CronJob specification", err.Error())
		return batch.CronJobSpec{}, diags
	}
	return expandCronJobSpec(ctx, value.(types.List), path.Root("spec"))
}

// cronJobSpecOps moves the live spec from previous to desired with a three-way
// strategic merge, and returns nothing when that leaves the live spec as it is.
func cronJobSpecOps(live *unstructured.Unstructured, previous, desired batch.CronJobSpec) (kubernetes.PatchOperations, error) {
	ops, err := common.StrategicMergeSpecOps(live, previous, desired, batch.CronJob{})
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		if replace, ok := op.(*kubernetes.ReplaceOperation); !ok || replace.Path != "/spec" || !sameJSON(replace.Value, live.Object["spec"]) {
			return ops, nil
		}
	}
	return nil, nil
}

func sameJSON(a, b any) bool {
	var x, y any
	dataA, errA := json.Marshal(a)
	dataB, errB := json.Marshal(b)
	return errA == nil && errB == nil && json.Unmarshal(dataA, &x) == nil && json.Unmarshal(dataB, &y) == nil &&
		reflect.DeepEqual(x, y)
}
