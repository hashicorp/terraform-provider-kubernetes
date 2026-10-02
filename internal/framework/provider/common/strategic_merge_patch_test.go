// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/applyconfigurations"
	"k8s.io/client-go/kubernetes/scheme"
	smdschema "sigs.k8s.io/structured-merge-diff/v4/schema"
)

// TestAmbiguousMergeKeyListsCoverWorkloadAPIs compares, for every list in the
// pod-template workload APIs, the strategic-merge patchMergeKey struct tag with
// the list-map keys the API server uses. Any list where they differ must be in
// AmbiguousMergeKeyLists, and every entry there must still exist.
func TestAmbiguousMergeKeyListsCoverWorkloadAPIs(t *testing.T) {
	resolver := applyconfigurations.NewTypeConverter(scheme.Scheme).TypeResolver
	found := map[string]string{}
	for name, example := range map[string]any{
		"io.k8s.api.apps.v1.Deployment":  appsv1.Deployment{},
		"io.k8s.api.apps.v1.DaemonSet":   appsv1.DaemonSet{},
		"io.k8s.api.apps.v1.StatefulSet": appsv1.StatefulSet{},
		"io.k8s.api.batch.v1.Job":        batchv1.Job{},
		"io.k8s.api.batch.v1.CronJob":    batchv1.CronJob{},
		"io.k8s.api.core.v1.Pod":         corev1.Pod{},
	} {
		parsed := resolver.Type(name)
		if parsed.Schema == nil {
			t.Fatalf("no structured-merge schema for %s", name)
		}
		walkMergeKeyLists(t, parsed.Schema, parsed.TypeRef, reflect.TypeOf(example), name, map[reflect.Type]bool{}, found)
	}
	for field, mergeKey := range found {
		if AmbiguousMergeKeyLists[field] != mergeKey {
			t.Errorf("list %q merges by %q but the API identifies entries by more keys; add it to AmbiguousMergeKeyLists", field, mergeKey)
		}
	}
	for field, mergeKey := range AmbiguousMergeKeyLists {
		if found[field] != mergeKey {
			t.Errorf("AmbiguousMergeKeyLists entry %q (%q) no longer matches any workload API list", field, mergeKey)
		}
	}
}

func walkMergeKeyLists(t *testing.T, s *smdschema.Schema, ref smdschema.TypeRef, goType reflect.Type, at string, seen map[reflect.Type]bool, found map[string]string) {
	t.Helper()
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	if goType.Kind() != reflect.Struct || seen[goType] {
		return
	}
	atom, ok := s.Resolve(ref)
	if !ok || atom.Map == nil {
		return
	}
	seen[goType] = true
	defer delete(seen, goType)
	for i := 0; i < goType.NumField(); i++ {
		field := goType.Field(i)
		if field.Anonymous {
			walkMergeKeyLists(t, s, ref, field.Type, at, seen, found)
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" || name == "status" {
			// Status is never written by the update paths.
			continue
		}
		structField, ok := atom.Map.FindField(name)
		if !ok {
			continue
		}
		if field.Type.Kind() != reflect.Slice || field.Type.Elem().Kind() == reflect.Uint8 {
			walkMergeKeyLists(t, s, structField.Type, field.Type, at+"."+name, seen, found)
			continue
		}
		list, ok := s.Resolve(structField.Type)
		if !ok || list.List == nil {
			continue
		}
		mergeKey := field.Tag.Get("patchMergeKey")
		if mergeKey != "" && strings.Contains(field.Tag.Get("patchStrategy"), "merge") &&
			list.List.ElementRelationship == smdschema.Associative &&
			!reflect.DeepEqual(list.List.Keys, []string{mergeKey}) {
			if previous, ok := found[name]; ok && previous != mergeKey {
				t.Errorf("list %q at %s merges by %q, elsewhere by %q", name, at, mergeKey, previous)
			}
			found[name] = mergeKey
		}
		walkMergeKeyLists(t, s, list.List.ElementType, field.Type.Elem(), at+"."+name+"[]", seen, found)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func applyStrategicPatch(t *testing.T, current corev1.PodSpec, patch []byte) corev1.PodSpec {
	t.Helper()
	patched, err := strategicpatch.StrategicMergePatch(mustJSON(t, current), patch, corev1.PodSpec{})
	if err != nil {
		t.Fatalf("applying %s: %v", patch, err)
	}
	var result corev1.PodSpec
	if err := json.Unmarshal(patched, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func podSpecWithPorts(ports ...corev1.ContainerPort) corev1.PodSpec {
	return corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "busybox", Ports: ports}}}
}

func TestTwoWayStrategicMergePatchReplacesAmbiguousLists(t *testing.T) {
	tcp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP, Name: "dns-tcp"}
	udp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP, Name: "dns-udp"}
	for name, tc := range map[string]struct{ before, after corev1.PodSpec }{
		"add":    {podSpecWithPorts(tcp), podSpecWithPorts(tcp, udp)},
		"remove": {podSpecWithPorts(tcp, udp), podSpecWithPorts(udp)},
		"clear":  {podSpecWithPorts(tcp, udp), podSpecWithPorts()},
	} {
		t.Run(name, func(t *testing.T) {
			patch, err := TwoWayStrategicMergePatch(mustJSON(t, tc.before), mustJSON(t, tc.after), corev1.PodSpec{})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(patch), "$setElementOrder/ports") {
				t.Errorf("patch orders ports by the ambiguous merge key: %s", patch)
			}
			// The server merges into its own copy, which may hold fields the
			// provider does not manage.
			live := tc.before
			live.HostUsers = new(bool)
			got := applyStrategicPatch(t, live, patch)
			if !reflect.DeepEqual(got.Containers[0].Ports, tc.after.Containers[0].Ports) {
				t.Errorf("ports = %#v, want %#v (patch %s)", got.Containers[0].Ports, tc.after.Containers[0].Ports, patch)
			}
			if got.HostUsers == nil {
				t.Errorf("unmanaged hostUsers was dropped by patch %s", patch)
			}
		})
	}
}

func TestTwoWayStrategicMergePatchLeavesUnchangedAmbiguousListsAlone(t *testing.T) {
	tcp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP}
	udp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP}
	before := podSpecWithPorts(tcp, udp)
	unchanged, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, before), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != "{}" {
		t.Fatalf("unchanged specification produced patch %s", unchanged)
	}

	after := before
	after.Containers = []corev1.Container{*before.Containers[0].DeepCopy()}
	after.Containers[0].Image = "busybox:new"
	patch, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, after), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(patch), "ports") {
		t.Fatalf("image-only change rewrote ports: %s", patch)
	}
}

func TestThreeWayStrategicMergeReplacesAmbiguousListsAndKeepsUnmanagedFields(t *testing.T) {
	zone := func(when corev1.UnsatisfiableConstraintAction, skew int32) corev1.TopologySpreadConstraint {
		return corev1.TopologySpreadConstraint{TopologyKey: "zone", WhenUnsatisfiable: when, MaxSkew: skew}
	}
	previous := podSpecWithPorts(corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP})
	previous.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{zone(corev1.DoNotSchedule, 1), zone(corev1.ScheduleAnyway, 2)}
	planned := *previous.DeepCopy()
	planned.Containers[0].Ports = append(planned.Containers[0].Ports, corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP})
	planned.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{zone(corev1.ScheduleAnyway, 3)}
	live := *previous.DeepCopy()
	live.HostUsers = new(bool)
	live.Containers[0].ResizePolicy = []corev1.ContainerResizePolicy{{ResourceName: corev1.ResourceCPU, RestartPolicy: corev1.NotRequired}}

	mergedJSON, err := ThreeWayStrategicMerge(mustJSON(t, previous), mustJSON(t, planned), mustJSON(t, live), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	var merged corev1.PodSpec
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged.Containers[0].Ports, planned.Containers[0].Ports) {
		t.Errorf("ports = %#v, want %#v", merged.Containers[0].Ports, planned.Containers[0].Ports)
	}
	if !reflect.DeepEqual(merged.TopologySpreadConstraints, planned.TopologySpreadConstraints) {
		t.Errorf("topology spread constraints = %#v, want %#v", merged.TopologySpreadConstraints, planned.TopologySpreadConstraints)
	}
	if merged.HostUsers == nil || len(merged.Containers[0].ResizePolicy) != 1 {
		t.Errorf("unmanaged live fields were dropped: hostUsers=%v resizePolicy=%#v", merged.HostUsers, merged.Containers[0].ResizePolicy)
	}
}

func TestStrategicMergeReplacesListsWithDuplicateMergeKeys(t *testing.T) {
	before := podSpecWithPorts()
	before.Containers[0].Env = []corev1.EnvVar{{Name: "MODE", Value: "a"}}
	after := *before.DeepCopy()
	after.Containers[0].Env = append(after.Containers[0].Env, corev1.EnvVar{Name: "MODE", Value: "b"})

	patch, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, after), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if got := applyStrategicPatch(t, before, patch); !reflect.DeepEqual(got.Containers[0].Env, after.Containers[0].Env) {
		t.Errorf("two-way env = %#v, want %#v (patch %s)", got.Containers[0].Env, after.Containers[0].Env, patch)
	}

	mergedJSON, err := ThreeWayStrategicMerge(mustJSON(t, after), mustJSON(t, before), mustJSON(t, after), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	var merged corev1.PodSpec
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged.Containers[0].Env, before.Containers[0].Env) {
		t.Errorf("three-way env = %#v, want %#v", merged.Containers[0].Env, before.Containers[0].Env)
	}
}
