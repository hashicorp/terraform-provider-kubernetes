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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/applyconfigurations"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	smdschema "sigs.k8s.io/structured-merge-diff/v4/schema"
)

// TestAmbiguousMergeKeyListsCoverWorkloadAPIs compares, for every list in the
// pod-template workload APIs, the strategic-merge patchMergeKey struct tag with
// the list-map keys the API server uses. Any list where they differ must be in
// ambiguousMergeKeyLists, and every entry there must still exist.
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
		if ambiguousMergeKeyLists[field] != mergeKey {
			t.Errorf("list %q merges by %q but the API identifies entries by more keys; add it to ambiguousMergeKeyLists", field, mergeKey)
		}
	}
	for field, mergeKey := range ambiguousMergeKeyLists {
		if found[field] != mergeKey {
			t.Errorf("ambiguousMergeKeyLists entry %q (%q) no longer matches any workload API list", field, mergeKey)
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
	withConstraints := func(s corev1.PodSpec, constraints ...corev1.TopologySpreadConstraint) corev1.PodSpec {
		s.TopologySpreadConstraints = constraints
		return s
	}
	zone := func(when corev1.UnsatisfiableConstraintAction, skew int32) corev1.TopologySpreadConstraint {
		return corev1.TopologySpreadConstraint{TopologyKey: "zone", WhenUnsatisfiable: when, MaxSkew: skew}
	}
	for name, tc := range map[string]struct{ before, after corev1.PodSpec }{
		"add":           {podSpecWithPorts(tcp), podSpecWithPorts(tcp, udp)},
		"add from none": {podSpecWithPorts(), podSpecWithPorts(tcp, udp)},
		"remove":        {podSpecWithPorts(tcp, udp), podSpecWithPorts(udp)},
		"clear":         {podSpecWithPorts(tcp, udp), podSpecWithPorts()},
		"constraints from none": {
			podSpecWithPorts(),
			withConstraints(podSpecWithPorts(), zone(corev1.DoNotSchedule, 1), zone(corev1.ScheduleAnyway, 2)),
		},
	} {
		t.Run(name, func(t *testing.T) {
			patch, err := TwoWayStrategicMergePatch(mustJSON(t, tc.before), mustJSON(t, tc.after), nil, corev1.PodSpec{})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(patch), "$setElementOrder/ports") ||
				strings.Contains(string(patch), "$setElementOrder/topologySpreadConstraints") {
				t.Errorf("patch orders a list by its ambiguous merge key: %s", patch)
			}
			// The server merges into its own copy, which may hold fields the
			// provider does not manage.
			live := tc.before
			live.HostUsers = new(bool)
			got := applyStrategicPatch(t, live, patch)
			if !reflect.DeepEqual(got.Containers[0].Ports, tc.after.Containers[0].Ports) {
				t.Errorf("ports = %#v, want %#v (patch %s)", got.Containers[0].Ports, tc.after.Containers[0].Ports, patch)
			}
			if !reflect.DeepEqual(got.TopologySpreadConstraints, tc.after.TopologySpreadConstraints) {
				t.Errorf("topology spread constraints = %#v, want %#v (patch %s)", got.TopologySpreadConstraints, tc.after.TopologySpreadConstraints, patch)
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
	unchanged, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, before), nil, corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != "{}" {
		t.Fatalf("unchanged specification produced patch %s", unchanged)
	}

	after := before
	after.Containers = []corev1.Container{*before.Containers[0].DeepCopy()}
	after.Containers[0].Image = "busybox:new"
	patch, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, after), nil, corev1.PodSpec{})
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

	patch, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, after), nil, corev1.PodSpec{})
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

func container(name, image string, ports ...corev1.ContainerPort) corev1.Container {
	return corev1.Container{Name: name, Image: image, Ports: ports}
}

func mergeThreeWay(t *testing.T, previous, planned, live corev1.PodSpec) corev1.PodSpec {
	t.Helper()
	mergedJSON, err := ThreeWayStrategicMerge(mustJSON(t, previous), mustJSON(t, planned), mustJSON(t, live), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mergedJSON), "$patch") || strings.Contains(string(mergedJSON), "$setElementOrder") {
		t.Fatalf("merged document kept patch directives: %s", mergedJSON)
	}
	var merged corev1.PodSpec
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		t.Fatal(err)
	}
	return merged
}

// TestThreeWayStrategicMergeFollowsLiveDrift covers the three-way merge used by
// the StatefulSet update when the live object no longer matches the prior
// state, as with -refresh=false or a saved plan applied after an out-of-band
// change. Ambiguous lists must end up exactly as planned, and no patch
// directive may leak into an entry that the merge appends whole.
func TestThreeWayStrategicMergeFollowsLiveDrift(t *testing.T) {
	tcp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP}
	udp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP}
	http := corev1.ContainerPort{ContainerPort: 80, Protocol: corev1.ProtocolTCP}
	spec := func(containers ...corev1.Container) corev1.PodSpec {
		return corev1.PodSpec{Containers: containers}
	}
	withEnv := func(c corev1.Container, env ...corev1.EnvVar) corev1.Container {
		c.Env = env
		return c
	}
	withDNS := func(s corev1.PodSpec, policy corev1.DNSPolicy) corev1.PodSpec {
		s.DNSPolicy = policy
		return s
	}
	modeA, modeB := corev1.EnvVar{Name: "MODE", Value: "a"}, corev1.EnvVar{Name: "MODE", Value: "b"}
	for name, tc := range map[string]struct {
		previous, planned, live corev1.PodSpec
	}{
		"managed container missing from live, ports unchanged": {
			previous: spec(container("main", "busybox", tcp)),
			planned:  spec(container("main", "busybox:new", tcp)),
			live:     spec(container("other", "busybox")),
		},
		"managed container missing from live, udp added": {
			previous: spec(container("main", "busybox", tcp)),
			planned:  spec(container("main", "busybox:new", tcp, udp)),
			live:     spec(container("other", "busybox")),
		},
		"sidecar removed out of band while main changes": {
			previous: spec(container("main", "busybox"), container("side", "busybox", http)),
			planned:  spec(container("main", "busybox:new"), container("side", "busybox", http)),
			live:     spec(container("main", "busybox")),
		},
		"container added out of band then in config": {
			previous: spec(container("main", "busybox")),
			planned:  spec(container("main", "busybox"), container("side", "busybox", tcp, udp)),
			live:     spec(container("main", "busybox"), container("side", "busybox", udp)),
		},
		"live lost a planned port": {
			previous: spec(container("main", "busybox", tcp, udp)),
			planned:  spec(container("main", "busybox:new", tcp, udp)),
			live:     spec(container("main", "busybox", tcp)),
		},
		"removal already applied out of band": {
			previous: spec(container("main", "busybox", tcp, udp)),
			planned:  spec(container("main", "busybox", tcp)),
			live:     spec(container("main", "busybox", tcp)),
		},
		"live gained a port, image changes": {
			previous: spec(container("main", "busybox", tcp)),
			planned:  spec(container("main", "busybox:new", tcp)),
			live:     spec(container("main", "busybox", tcp, udp)),
		},
		"live gained a port, only a pod field changes": {
			previous: spec(container("main", "busybox", tcp)),
			planned:  withDNS(spec(container("main", "busybox", tcp)), corev1.DNSDefault),
			live:     spec(container("main", "busybox", tcp, udp)),
		},
		"live gained a duplicate env": {
			previous: spec(withEnv(container("main", "busybox"), modeA)),
			planned:  spec(withEnv(container("main", "busybox:new"), modeA)),
			live:     spec(withEnv(container("main", "busybox"), modeA, modeB)),
		},
	} {
		t.Run(name, func(t *testing.T) {
			// Containers only live has are not managed and stay as they are.
			merged := mergeThreeWay(t, tc.previous, tc.planned, tc.live)
			for _, want := range tc.planned.Containers {
				var got *corev1.Container
				for i := range merged.Containers {
					if merged.Containers[i].Name == want.Name {
						got = &merged.Containers[i]
					}
				}
				if got == nil {
					t.Fatalf("container %q missing from %#v", want.Name, merged.Containers)
				}
				if got.Image != want.Image || !reflect.DeepEqual(got.Ports, want.Ports) || !reflect.DeepEqual(got.Env, want.Env) {
					t.Errorf("container %q = image %q ports %#v env %#v, want image %q ports %#v env %#v",
						want.Name, got.Image, got.Ports, got.Env, want.Image, want.Ports, want.Env)
				}
			}
			if merged.DNSPolicy != tc.planned.DNSPolicy {
				t.Errorf("dnsPolicy = %q, want %q", merged.DNSPolicy, tc.planned.DNSPolicy)
			}
		})
	}
}

// TestThreeWayStrategicMergeRewritesDriftUnderTemplate checks an ambiguous
// list that drifted live while only the template labels change.
func TestThreeWayStrategicMergeRewritesDriftUnderTemplate(t *testing.T) {
	previous, planned, live := portsTemplate("1", tcp53), portsTemplate("2", tcp53), portsTemplate("1", tcp53, udp53)
	mergedJSON, err := ThreeWayStrategicMerge(mustJSON(t, previous), mustJSON(t, planned), mustJSON(t, live), corev1.PodTemplateSpec{})
	if err != nil {
		t.Fatal(err)
	}
	var merged corev1.PodTemplateSpec
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged.Spec.Containers[0].Ports, planned.Spec.Containers[0].Ports) || merged.Labels["rev"] != "2" {
		t.Errorf("merged = %s, want ports %#v and label rev=2", mergedJSON, planned.Spec.Containers[0].Ports)
	}
}

// TestReplaceAmbiguousListsAddsMissingPatchEntries feeds the rewrite a base
// patch that has no entry for the template spec or the container. Patches from
// strategicpatch carry at least a $setElementOrder entry there today, so this
// pins the fallback that adds them should that change.
func TestReplaceAmbiguousListsAddsMissingPatchEntries(t *testing.T) {
	previous, planned, live := portsTemplate("1", tcp53), portsTemplate("1", tcp53), portsTemplate("1", tcp53, udp53)
	meta, err := strategicpatch.NewPatchMetaFromStruct(corev1.PodTemplateSpec{})
	if err != nil {
		t.Fatal(err)
	}
	patch, err := replaceAmbiguousLists([]byte("{}"), mustJSON(t, previous), mustJSON(t, planned), mustJSON(t, live), true, meta)
	if err != nil {
		t.Fatal(err)
	}
	mergedJSON, err := strategicpatch.StrategicMergePatch(mustJSON(t, live), patch, corev1.PodTemplateSpec{})
	if err != nil {
		t.Fatalf("applying %s: %v", patch, err)
	}
	var merged corev1.PodTemplateSpec
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged.Spec.Containers[0].Ports, planned.Spec.Containers[0].Ports) {
		t.Errorf("patch %s left ports %#v, want %#v", patch, merged.Spec.Containers[0].Ports, planned.Spec.Containers[0].Ports)
	}
}

var (
	tcp53 = corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP}
	udp53 = corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP}
)

func portsTemplate(revision string, ports ...corev1.ContainerPort) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"rev": revision}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{container("main", "busybox", ports...)}},
	}
}

// TestTwoWayStrategicMergePatchUsesLiveEntries covers an entry that the prior
// state lacks but the server already holds, for example a container added out
// of band and then added in configuration. The server merges the patch into
// its entry, so the ambiguous lists inside it must be replaced, not merged.
func TestTwoWayStrategicMergePatchUsesLiveEntries(t *testing.T) {
	tcp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP}
	udp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP}
	previous := corev1.PodSpec{Containers: []corev1.Container{container("main", "busybox")}}
	planned := corev1.PodSpec{Containers: []corev1.Container{container("main", "busybox"), container("side", "busybox", tcp, udp)}}
	live := corev1.PodSpec{Containers: []corev1.Container{container("main", "busybox"), container("side", "busybox", udp)}}

	patch, err := TwoWayStrategicMergePatch(mustJSON(t, previous), mustJSON(t, planned), mustJSON(t, live), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	got := applyStrategicPatch(t, live, patch)
	if len(got.Containers) != 2 || !reflect.DeepEqual(got.Containers[1].Ports, planned.Containers[1].Ports) {
		t.Errorf("containers = %#v, want side ports %#v (patch %s)", got.Containers, planned.Containers[1].Ports, patch)
	}

	// When the server lacks the entry, the patch appends it verbatim and must
	// not carry a directive inside it.
	patch, err = TwoWayStrategicMergePatch(mustJSON(t, previous), mustJSON(t, planned), mustJSON(t, previous), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	got = applyStrategicPatch(t, previous, patch)
	if len(got.Containers) != 2 || !reflect.DeepEqual(got.Containers[1].Ports, planned.Containers[1].Ports) {
		t.Errorf("containers = %#v, want side ports %#v (patch %s)", got.Containers, planned.Containers[1].Ports, patch)
	}
}

// TestStrategicMergePatchKeepsLargeIntegers checks that rewriting the patch
// does not round int64 values through float64.
func TestStrategicMergePatchKeepsLargeIntegers(t *testing.T) {
	const large = int64(1<<53 + 1)
	tcp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolTCP}
	udp := corev1.ContainerPort{ContainerPort: 53, Protocol: corev1.ProtocolUDP}
	before := podSpecWithPorts(tcp)
	after := podSpecWithPorts(tcp, udp)
	after.TerminationGracePeriodSeconds = ptr.To(large)

	patch, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, after), nil, corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if got := applyStrategicPatch(t, before, patch); got.TerminationGracePeriodSeconds == nil || *got.TerminationGracePeriodSeconds != large {
		t.Errorf("two-way patch %s lost precision", patch)
	}
	merged := mergeThreeWay(t, before, after, before)
	if merged.TerminationGracePeriodSeconds == nil || *merged.TerminationGracePeriodSeconds != large {
		t.Errorf("three-way merge lost precision: %v", merged.TerminationGracePeriodSeconds)
	}
}

// TestTwoWayStrategicMergePatchReturnsUntouchedPatchVerbatim checks that a
// patch without ambiguous lists is exactly what strategicpatch produced.
func TestTwoWayStrategicMergePatchReturnsUntouchedPatchVerbatim(t *testing.T) {
	before := podSpecWithPorts()
	after := podSpecWithPorts()
	after.Containers[0].Image = "busybox:new"
	after.TerminationGracePeriodSeconds = ptr.To(int64(1<<53 + 1))
	want, err := strategicpatch.CreateTwoWayMergePatch(mustJSON(t, before), mustJSON(t, after), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := TwoWayStrategicMergePatch(mustJSON(t, before), mustJSON(t, after), mustJSON(t, before), corev1.PodSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("patch = %s, want %s", got, want)
	}
}
