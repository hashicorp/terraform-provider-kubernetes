// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type servicePatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	From  string `json:"from,omitempty"`
	Value any    `json:"value,omitempty"`
}

func serviceMetadataPatchOps(state, plan common.NamespacedMetadataModel, live metav1.ObjectMeta) []servicePatchOperation {
	var ops []servicePatchOperation
	ops = append(ops, serviceMapPatchOps("/metadata/annotations", state.Annotations, plan.Annotations, live.Annotations)...)
	ops = append(ops, serviceMapPatchOps("/metadata/labels", state.Labels, plan.Labels, live.Labels)...)
	return ops
}

func serviceMapPatchOps(path string, state, plan types.Map, live map[string]string) []servicePatchOperation {
	if plan.IsUnknown() || state.Equal(plan) {
		return nil
	}
	oldValues, values := common.ExpandMapForPatch(state), common.ExpandMapForPatch(plan)
	var ops []servicePatchOperation
	if len(live) == 0 && len(values) > 0 {
		// The update includes a resourceVersion test, so a concurrently created
		// controller map cannot be overwritten by this initialization.
		ops = append(ops, servicePatchOperation{Op: "add", Path: path, Value: map[string]string{}})
	}
	keys := make([]string, 0, len(oldValues)+len(values))
	for key := range oldValues {
		keys = append(keys, key)
	}
	for key := range values {
		if _, found := oldValues[key]; !found {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, wanted := values[key]
		actual, exists := live[key]
		if !wanted {
			if exists {
				ops = append(ops, servicePatchOperation{Op: "remove", Path: path + "/" + serviceJSONPointer(key)})
			}
			continue
		}
		if exists && actual == value {
			continue
		}
		ops = append(ops, servicePatchOperation{Op: "add", Path: path + "/" + serviceJSONPointer(key), Value: value})
	}
	return ops
}

func serviceJSONPointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func serviceSpecPatchOps(ctx context.Context, state, plan ServiceV1SpecModel, live corev1.ServiceSpec) ([]servicePatchOperation, diag.Diagnostics) {
	prior, diags := expandServiceSpec(ctx, state)
	desired, d := expandServiceSpec(ctx, plan)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	var ops []servicePatchOperation
	add := func(field string, value any) {
		ops = append(ops, servicePatchOperation{Op: "add", Path: "/spec/" + field, Value: value})
	}
	remove := func(field string) {
		ops = append(ops, servicePatchOperation{Op: "remove", Path: "/spec/" + field})
	}
	stringChange := func(field string, old, next types.String) {
		if next.IsUnknown() || next.ValueString() == old.ValueString() {
			return
		}
		add(field, next.ValueString())
	}
	computedStringChange := func(field string, old, next types.String) {
		if next.IsNull() {
			return
		}
		stringChange(field, old, next)
	}
	stringChange("externalName", state.ExternalName, plan.ExternalName)
	stringChange("loadBalancerIP", state.LoadBalancerIP, plan.LoadBalancerIP)
	computedStringChange("externalTrafficPolicy", state.ExternalTrafficPolicy, plan.ExternalTrafficPolicy)
	computedStringChange("internalTrafficPolicy", state.InternalTrafficPolicy, plan.InternalTrafficPolicy)
	computedStringChange("ipFamilyPolicy", state.IPFamilyPolicy, plan.IPFamilyPolicy)
	if !plan.IPFamilies.IsUnknown() && !plan.IPFamilies.IsNull() && !slices.Equal(prior.IPFamilies, desired.IPFamilies) {
		add("ipFamilies", desired.IPFamilies)
	}
	if !plan.ExternalIPs.IsUnknown() && !serviceStringSlicesEqual(prior.ExternalIPs, desired.ExternalIPs) {
		add("externalIPs", serviceNonNilStrings(desired.ExternalIPs))
	}
	if !plan.LoadBalancerSourceRanges.IsUnknown() && !serviceStringSlicesEqual(prior.LoadBalancerSourceRanges, desired.LoadBalancerSourceRanges) {
		add("loadBalancerSourceRanges", serviceNonNilStrings(desired.LoadBalancerSourceRanges))
	}
	if !plan.Selector.IsUnknown() && !serviceStringMapsEqual(prior.Selector, desired.Selector) {
		// Selector is an owned field, unlike metadata's partially owned maps.
		selector := desired.Selector
		if selector == nil {
			selector = map[string]string{}
		}
		add("selector", selector)
	}
	if !plan.PublishNotReadyAddresses.IsUnknown() && prior.PublishNotReadyAddresses != desired.PublishNotReadyAddresses {
		add("publishNotReadyAddresses", desired.PublishNotReadyAddresses)
	}
	typeChanged := prior.Type != desired.Type && !plan.Type.IsUnknown()
	if desired.Type == corev1.ServiceTypeLoadBalancer && (typeChanged || prior.AllocateLoadBalancerNodePorts != nil && desired.AllocateLoadBalancerNodePorts != nil && *prior.AllocateLoadBalancerNodePorts != *desired.AllocateLoadBalancerNodePorts) {
		if desired.AllocateLoadBalancerNodePorts != nil {
			add("allocateLoadBalancerNodePorts", *desired.AllocateLoadBalancerNodePorts)
		}
	}
	affinityChanged := !plan.SessionAffinity.IsUnknown() && desired.SessionAffinity != prior.SessionAffinity
	if affinityChanged {
		add("sessionAffinity", desired.SessionAffinity)
		if desired.SessionAffinity == corev1.ServiceAffinityNone && live.SessionAffinityConfig != nil {
			remove("sessionAffinityConfig")
		}
	}
	if desired.SessionAffinity != corev1.ServiceAffinityNone && !plan.SessionAffinityConfig.IsUnknown() && !plan.SessionAffinityConfig.IsNull() {
		ops = append(ops, serviceAffinityPatchOps(prior.SessionAffinityConfig, desired.SessionAffinityConfig, live.SessionAffinityConfig)...)
	}
	ops = append(ops, servicePortsPatchOps(state.Ports, plan.Ports, live.Ports, typeChanged && (desired.Type == corev1.ServiceTypeClusterIP || desired.Type == corev1.ServiceTypeExternalName))...)
	if typeChanged {
		if desired.Type == corev1.ServiceTypeExternalName {
			// These allocations belong to Kubernetes. Clear them only for the
			// explicit type transition that makes them invalid.
			if live.ClusterIP != "" {
				remove("clusterIP")
			}
			if len(live.ClusterIPs) > 0 {
				remove("clusterIPs")
			}
			if len(live.IPFamilies) > 0 {
				remove("ipFamilies")
			}
			if live.IPFamilyPolicy != nil {
				remove("ipFamilyPolicy")
			}
			if live.InternalTrafficPolicy != nil {
				remove("internalTrafficPolicy")
			}
		}
		if desired.Type != corev1.ServiceTypeLoadBalancer {
			if live.AllocateLoadBalancerNodePorts != nil {
				remove("allocateLoadBalancerNodePorts")
			}
			if live.LoadBalancerClass != nil {
				remove("loadBalancerClass")
			}
		}
		if desired.Type == corev1.ServiceTypeClusterIP || desired.Type == corev1.ServiceTypeExternalName {
			hasExternalIPs := len(desired.ExternalIPs) > 0
			if plan.ExternalIPs.IsUnknown() {
				hasExternalIPs = len(live.ExternalIPs) > 0
			}
			if live.ExternalTrafficPolicy != "" && (desired.Type == corev1.ServiceTypeExternalName || !hasExternalIPs) {
				remove("externalTrafficPolicy")
			}
			if live.HealthCheckNodePort != 0 {
				remove("healthCheckNodePort")
			}
		}
		add("type", desired.Type)
	}
	return ops, diags
}

func serviceAffinityPatchOps(prior, desired, live *corev1.SessionAffinityConfig) []servicePatchOperation {
	// An omitted/unknown timeout delegates ownership to the API. Merely
	// resolving its parent collections must not trigger an API mutation.
	if desired == nil || desired.ClientIP == nil || desired.ClientIP.TimeoutSeconds == nil {
		return nil
	}
	if prior != nil && prior.ClientIP != nil && prior.ClientIP.TimeoutSeconds != nil && *prior.ClientIP.TimeoutSeconds == *desired.ClientIP.TimeoutSeconds {
		return nil
	}
	var ops []servicePatchOperation
	if live == nil {
		ops = append(ops, servicePatchOperation{Op: "add", Path: "/spec/sessionAffinityConfig", Value: map[string]any{}})
	}
	if live == nil || live.ClientIP == nil {
		ops = append(ops, servicePatchOperation{Op: "add", Path: "/spec/sessionAffinityConfig/clientIP", Value: map[string]any{}})
	}
	ops = append(ops, servicePatchOperation{Op: "add", Path: "/spec/sessionAffinityConfig/clientIP/timeoutSeconds", Value: *desired.ClientIP.TimeoutSeconds})
	return ops
}

func servicePortsPatchOps(state, plan []ServiceV1PortModel, live []corev1.ServicePort, removeNodePorts bool) []servicePatchOperation {
	if !removeNodePorts && !servicePortsChanged(state, plan) {
		return nil
	}
	var ops []servicePatchOperation
	previous := make([]corev1.ServicePort, len(state))
	desiredPorts := make([]corev1.ServicePort, len(plan))
	for i, port := range state {
		previous[i] = expandServicePort(port)
	}
	for i, port := range plan {
		desiredPorts[i] = expandServicePort(port)
	}
	planToState := serviceCorrelatePorts(previous, desiredPorts)
	stateToLive := serviceCorrelatePorts(live, previous)
	planToLive := serviceUnmatchedPorts(len(plan))
	claimedLive := make([]bool, len(live))
	ownedLive := make([]bool, len(live))
	for _, index := range stateToLive {
		if index >= 0 {
			ownedLive[index] = true
		}
	}
	for i, previousIndex := range planToState {
		if previousIndex >= 0 && stateToLive[previousIndex] >= 0 {
			planToLive[i] = stateToLive[previousIndex]
			claimedLive[planToLive[i]] = true
		}
	}
	// Reserve all existing owned-port matches before adopting any matching
	// unowned API entries. A new first port must not steal a later port.
	serviceMatchStablePorts(live, desiredPorts, planToLive, claimedLive)
	current := slices.Clone(live)
	currentIDs := make([]int, len(live))
	for i := range currentIDs {
		currentIDs[i] = i
	}
	if len(current) == 0 && len(plan) > 0 {
		ops = append(ops, servicePatchOperation{Op: "add", Path: "/spec/ports", Value: []corev1.ServicePort{}})
	}
	for i, planned := range plan {
		desired := desiredPorts[i]
		j := -1
		if planToLive[i] >= 0 {
			j = slices.Index(currentIDs, planToLive[i])
		}
		base := fmt.Sprintf("/spec/ports/%d", i)
		if j < 0 {
			if removeNodePorts {
				desired.NodePort = 0
			}
			ops = append(ops, servicePatchOperation{Op: "add", Path: base, Value: desired})
			current = slices.Insert(current, i, desired)
			currentIDs = slices.Insert(currentIDs, i, -1)
			continue
		}
		if j != i {
			// A JSON Patch move preserves future API fields on the port object.
			ops = append(ops, servicePatchOperation{Op: "move", Path: base, From: fmt.Sprintf("/spec/ports/%d", j)})
			moved := current[j]
			movedID := currentIDs[j]
			current = slices.Delete(current, j, j+1)
			current = slices.Insert(current, i, moved)
			currentIDs = slices.Delete(currentIDs, j, j+1)
			currentIDs = slices.Insert(currentIDs, i, movedID)
		}
		var prior corev1.ServicePort
		if previousIndex := planToState[i]; previousIndex >= 0 {
			prior = previous[previousIndex]
		} else {
			prior = current[i]
		}
		add := func(field string, value any) {
			ops = append(ops, servicePatchOperation{Op: "add", Path: base + "/" + field, Value: value})
		}
		if desired.Name != prior.Name {
			add("name", desired.Name)
		}
		if desired.Port != prior.Port {
			add("port", desired.Port)
		}
		if desired.Protocol != prior.Protocol && !planned.Protocol.IsUnknown() {
			add("protocol", desired.Protocol)
		}
		if !planned.TargetPort.IsUnknown() && !planned.TargetPort.IsNull() && desired.TargetPort != prior.TargetPort {
			add("targetPort", desired.TargetPort)
		}
		if !planned.NodePort.IsUnknown() && !planned.NodePort.IsNull() && desired.NodePort != prior.NodePort && !removeNodePorts {
			add("nodePort", desired.NodePort)
		}
		if removeNodePorts && current[i].NodePort != 0 {
			ops = append(ops, servicePatchOperation{Op: "remove", Path: base + "/nodePort"})
		}
		if ptr.Deref(desired.AppProtocol, "") != ptr.Deref(prior.AppProtocol, "") {
			if desired.AppProtocol != nil {
				add("appProtocol", *desired.AppProtocol)
			} else if current[i].AppProtocol != nil {
				ops = append(ops, servicePatchOperation{Op: "remove", Path: base + "/appProtocol"})
			}
		}
	}
	for i := len(current) - 1; i >= len(plan); i-- {
		if currentIDs[i] >= 0 && ownedLive[currentIDs[i]] {
			ops = append(ops, servicePatchOperation{Op: "remove", Path: fmt.Sprintf("/spec/ports/%d", i)})
		} else if removeNodePorts && current[i].NodePort != 0 {
			ops = append(ops, servicePatchOperation{Op: "remove", Path: fmt.Sprintf("/spec/ports/%d/nodePort", i)})
		}
	}
	return ops
}

func servicePortsChanged(state, plan []ServiceV1PortModel) bool {
	if len(state) != len(plan) {
		return true
	}
	for i, planned := range plan {
		old, next := expandServicePort(state[i]), expandServicePort(planned)
		if old.Name != next.Name || old.Port != next.Port || old.Protocol != next.Protocol || ptr.Deref(old.AppProtocol, "") != ptr.Deref(next.AppProtocol, "") {
			return true
		}
		if !planned.NodePort.IsNull() && !planned.NodePort.IsUnknown() && old.NodePort != next.NodePort {
			return true
		}
		if !planned.TargetPort.IsNull() && !planned.TargetPort.IsUnknown() && old.TargetPort != next.TargetPort {
			return true
		}
	}
	return false
}

func serviceUnmatchedPorts(count int) []int {
	matches := make([]int, count)
	for i := range matches {
		matches[i] = -1
	}
	return matches
}

func serviceCorrelatePorts(previous, planned []corev1.ServicePort) []int {
	matches := serviceUnmatchedPorts(len(planned))
	claimed := make([]bool, len(previous))
	serviceMatchStablePorts(previous, planned, matches, claimed)
	if len(previous) != len(planned) {
		return matches
	}
	oldIndex, newIndex := -1, -1
	oldCount, newCount := 0, 0
	for i, used := range claimed {
		if !used {
			oldIndex = i
			oldCount++
		}
	}
	for i, match := range matches {
		if match < 0 {
			newIndex = i
			newCount++
		}
	}
	// Once all stable matches are reserved, one remaining pair at unchanged
	// cardinality is unambiguous, even if both its name and number changed.
	if oldCount == 1 && newCount == 1 {
		matches[newIndex] = oldIndex
	}
	return matches
}

func serviceMatchStablePorts(previous, planned []corev1.ServicePort, matches []int, claimed []bool) {
	for _, byName := range []bool{true, false} {
		for i, port := range planned {
			if matches[i] >= 0 || byName && port.Name == "" {
				continue
			}
			match := -1
			for j, candidate := range previous {
				if claimed[j] {
					continue
				}
				same := port.Port == candidate.Port && port.Protocol == candidate.Protocol
				if byName {
					same = port.Name == candidate.Name
				}
				if !same {
					continue
				}
				if match >= 0 {
					match = -1
					break
				}
				match = j
			}
			if match >= 0 {
				matches[i] = match
				claimed[match] = true
			}
		}
	}
}

func serviceAPIPortIndex(ports []corev1.ServicePort, target corev1.ServicePort, start int) int {
	for i := start; i < len(ports); i++ {
		if target.Name != "" && target.Name == ports[i].Name {
			return i
		}
	}
	for i := start; i < len(ports); i++ {
		if target.Port == ports[i].Port && target.Protocol == ports[i].Protocol {
			return i
		}
	}
	// Kubernetes permits an unnamed port only on a single-port Service. Its
	// allocation remains the same port when its number or protocol is edited.
	if start == 0 && len(ports) == 1 && ports[0].Name == "" && target.Name == "" {
		return 0
	}
	return -1
}

func serviceStringSlicesEqual(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	sort.Strings(a)
	sort.Strings(b)
	return slices.Equal(a, b)
}

func serviceNonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func serviceStringMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if other, exists := b[k]; !exists || other != v {
			return false
		}
	}
	return true
}
