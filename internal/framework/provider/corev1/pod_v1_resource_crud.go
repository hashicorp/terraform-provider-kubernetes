// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	k8sretry "k8s.io/client-go/util/retry"
)

const podDefaultTimeout = 5 * time.Minute

func (p *PodV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PodV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, podDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	clients, _, diags := p.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	spec, diags := podV1Spec().ExpandSpec(ctx, plan.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	targets, diags := podV1TargetStates(ctx, plan.TargetState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pod, err := conn.CoreV1().Pods(metadata.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metadata,
		Spec:       spec,
	}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Pod", err.Error())
		return
	}

	// Save the returned identity before flattening or waiting can fail.
	plan.ID = types.StringValue(kubernetes.BuildId(pod.ObjectMeta))
	podV1SetMetadata(&plan, pod.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, podV1Identity(pod.Namespace, pod.Name))...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Both reads are flattened against the plan, not against each other.
	planned := plan.Spec
	plan.Spec, diags = podV1Spec().FlattenSpec(ctx, pod.Spec, planned, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waiter := retry.StateChangeConf{
		Target:  targets,
		Pending: []string{string(corev1.PodPending)},
		Timeout: timeout,
		Refresh: func() (interface{}, string, error) {
			current, err := conn.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if err != nil {
				return nil, "", err
			}
			return current, string(current.Status.Phase), nil
		},
	}
	result, err := waiter.WaitForStateContext(ctx)
	if err != nil {
		// Preserve the wait failure even if the optional event lookup is denied or
		// the operation deadline has expired.
		warnings, warningErr := kubernetes.GetLastWarningsForObject(ctx, conn, pod.ObjectMeta, podKind, 3)
		detail := err.Error()
		if warningErr != nil {
			detail += fmt.Sprintf("\nCould not retrieve Pod warnings: %s", warningErr)
		} else {
			detail += kubernetes.StringifyEvents(warnings)
		}
		resp.Diagnostics.AddError("Error waiting for Pod", detail)
		return
	}
	current := result.(*corev1.Pod)
	plan.Spec, diags = podV1Spec().FlattenSpec(ctx, current.Spec, planned, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	podV1SetMetadata(&plan, current.ObjectMeta)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (p *PodV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PodV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	clients, filters, diags := p.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Pod ID", err.Error())
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	pod, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		tflog.Info(ctx, "Pod no longer exists", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading Pod", err.Error())
		return
	}
	state.Metadata, diags = common.FlattenNamespacedMetadata(ctx, pod.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	state.Spec, diags = podV1Spec().FlattenSpec(ctx, pod.Spec, state.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.TargetState.IsNull() {
		state.TargetState = types.ListValueMust(types.StringType, nil)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, podV1Identity(namespace, name))...)
}

func (p *PodV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PodV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	clients, _, diags := p.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Pod ID", err.Error())
		return
	}
	priorMetadata, diags := common.ExpandNamespacedMetadata(ctx, state.Metadata)
	resp.Diagnostics.Append(diags...)
	plannedMetadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	priorSpec, diags := podV1Spec().ExpandSpec(ctx, state.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	plannedSpec, diags := podV1Spec().ExpandSpec(ctx, plan.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	var pod *corev1.Pod
	err = k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		current, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		patch := podV1UpdatePatch(priorMetadata, plannedMetadata, &priorSpec, &plannedSpec, current)
		if len(patch) == 0 {
			pod = current
			return nil
		}
		// A resourceVersion precondition protects concurrent controller metadata
		// edits; strategic merge updates containers by name, not by API list index.
		metadata, ok := patch["metadata"].(map[string]interface{})
		if !ok {
			metadata = map[string]interface{}{}
			patch["metadata"] = metadata
		}
		metadata["resourceVersion"] = current.ResourceVersion
		data, err := json.Marshal(patch)
		if err != nil {
			return err
		}
		pod, err = conn.CoreV1().Pods(namespace).Patch(ctx, name, k8stypes.StrategicMergePatchType, data, metav1.PatchOptions{})
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Pod", err.Error())
		return
	}

	plan.ID = state.ID
	podV1SetMetadata(&plan, pod.ObjectMeta)
	plan.Spec, diags = podV1Spec().FlattenSpec(ctx, pod.Spec, plan.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, podV1Identity(namespace, name))...)
}

func (p *PodV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PodV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := state.Timeouts.Delete(ctx, podDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	clients, _, diags := p.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Pod ID", err.Error())
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	err = conn.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Pod", err.Error())
		return
	}
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		pod, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("Pod %s/%s still exists (%s)", namespace, name, pod.Status.Phase))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Pod deletion", err.Error())
	}
}

func podV1SetMetadata(model *PodV1Model, metadata metav1.ObjectMeta) {
	if len(model.Metadata) == 0 {
		model.Metadata = []common.NamespacedMetadataModel{{}}
	}
	model.Metadata[0].Name = types.StringValue(metadata.Name)
	model.Metadata[0].Namespace = types.StringValue(metadata.Namespace)
	model.Metadata[0].UID = types.StringValue(string(metadata.UID))
	model.Metadata[0].Generation = types.Int64Value(metadata.Generation)
	model.Metadata[0].ResourceVersion = types.StringValue(metadata.ResourceVersion)
}

func podV1TargetStates(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	if value.IsNull() || (!value.IsUnknown() && len(value.Elements()) == 0) {
		return []string{string(corev1.PodRunning)}, nil
	}
	var targets []string
	diags := value.ElementsAs(ctx, &targets, false)
	return targets, diags
}

func podV1UpdatePatch(priorMeta, plannedMeta metav1.ObjectMeta, priorSpec, plannedSpec *corev1.PodSpec, live *corev1.Pod) map[string]interface{} {
	patch := map[string]interface{}{}
	metadata := map[string]interface{}{}
	if labels := podV1MetadataChanges(priorMeta.Labels, plannedMeta.Labels, live.Labels); len(labels) != 0 {
		metadata["labels"] = labels
	}
	if annotations := podV1MetadataChanges(priorMeta.Annotations, plannedMeta.Annotations, live.Annotations); len(annotations) != 0 {
		metadata["annotations"] = annotations
	}
	if len(metadata) != 0 {
		patch["metadata"] = metadata
	}
	spec := map[string]interface{}{}
	if !reflect.DeepEqual(priorSpec.ActiveDeadlineSeconds, plannedSpec.ActiveDeadlineSeconds) &&
		!reflect.DeepEqual(live.Spec.ActiveDeadlineSeconds, plannedSpec.ActiveDeadlineSeconds) {
		spec["activeDeadlineSeconds"] = plannedSpec.ActiveDeadlineSeconds
	}
	priorImages := make(map[string]string, len(priorSpec.Containers))
	liveImages := make(map[string]string, len(live.Spec.Containers))
	for _, c := range priorSpec.Containers {
		priorImages[c.Name] = c.Image
	}
	for _, c := range live.Spec.Containers {
		liveImages[c.Name] = c.Image
	}
	var containers []map[string]string
	for _, c := range plannedSpec.Containers {
		if priorImages[c.Name] != c.Image && liveImages[c.Name] != c.Image {
			containers = append(containers, map[string]string{"name": c.Name, "image": c.Image})
		}
	}
	if len(containers) != 0 {
		spec["containers"] = containers
	}
	if len(spec) != 0 {
		patch["spec"] = spec
	}
	return patch
}

func podV1MetadataChanges(prior, plan, live map[string]string) map[string]interface{} {
	changes := map[string]interface{}{}
	for key := range prior {
		if _, retained := plan[key]; !retained {
			if _, exists := live[key]; exists {
				changes[key] = nil
			}
		}
	}
	for key, value := range plan {
		if current, exists := live[key]; !exists || current != value {
			changes[key] = value
		}
	}
	return changes
}
