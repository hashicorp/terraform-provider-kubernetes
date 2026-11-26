// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"reflect"
	"sort"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
)

func decodeJSON(t *testing.T, text string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// The manifest below, and the object and managed fields the API server keeps
// for it after a server-side apply. Its empty list and empty map are not in
// the object: a client that rewrites the object leaves them out.
const (
	testManifest = `{"apiVersion":"apps/v1","kind":"Deployment",
		"metadata":{"name":"web","labels":{"app":"web","tier":"front"},"finalizers":["keep"]},
		"spec":{"replicas":2,"selector":{"matchLabels":{"app":"web"}},"template":{"spec":{"tolerations":[],"nodeSelector":{},"containers":[
			{"name":"web","image":"nginx:1","args":["--a"],"ports":[{"containerPort":80}],"resources":{"requests":{"cpu":0.5,"memory":"1024Mi"}}}]}}}}`
	testLive = `{"apiVersion":"apps/v1","kind":"Deployment",
		"metadata":{"name":"web","namespace":"default","uid":"1","labels":{"app":"web","tier":"front"},"annotations":{"other":"x"},"finalizers":["keep","theirs"]},
		"spec":{"replicas":2,"revisionHistoryLimit":10,"selector":{"matchLabels":{"app":"web"}},"template":{"spec":{"dnsPolicy":"ClusterFirst","containers":[
			{"name":"sidecar","image":"proxy:1"},
			{"name":"web","image":"nginx:1","args":["--a"],"imagePullPolicy":"IfNotPresent","ports":[{"containerPort":80,"protocol":"TCP"}],"resources":{"requests":{"cpu":"500m","memory":"1024Mi"}}}]}}},
		"status":{"replicas":2}}`
	testFields = `{"f:metadata":{"f:labels":{"f:app":{},"f:tier":{}},"f:finalizers":{"v:\"keep\"":{}}},
		"f:spec":{"f:replicas":{},"f:selector":{},"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"web\"}":{".":{},"f:name":{},"f:image":{},"f:args":{},
			"f:ports":{"k:{\"containerPort\":80,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{}}},"f:resources":{"f:requests":{"f:cpu":{},"f:memory":{}}}}}}}}}`
	// The same fields without the ones other clients took over.
	testFieldsAfterChanges = `{"f:metadata":{"f:labels":{"f:app":{}}},
		"f:spec":{"f:selector":{},"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"web\"}":{".":{},"f:name":{},
			"f:ports":{"k:{\"containerPort\":80,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{}}},"f:resources":{"f:requests":{"f:cpu":{},"f:memory":{}}}}}}}}}`
	testOtherFields = `{"f:metadata":{"f:finalizers":{"v:\"theirs\"":{}}},"f:spec":{"f:replicas":{},"f:template":{"f:spec":{"f:containers":{
		"k:{\"name\":\"sidecar\"}":{".":{},"f:name":{},"f:image":{}},"k:{\"name\":\"web\"}":{"f:image":{},"f:args":{}}}}}}}`
)

func liveObject(t *testing.T, edit func(*unstructured.Unstructured), entries ...metav1.ManagedFieldsEntry) *unstructured.Unstructured {
	t.Helper()
	obj := &unstructured.Unstructured{Object: decodeJSON(t, testLive)}
	if edit != nil {
		edit(obj)
	}
	obj.SetManagedFields(entries)
	return obj
}

func entry(manager string, operation metav1.ManagedFieldsOperationType, fields string) metav1.ManagedFieldsEntry {
	return metav1.ManagedFieldsEntry{Manager: manager, Operation: operation, FieldsV1: metav1.NewFieldsV1(fields)}
}

func TestLiveChanges(t *testing.T) {
	changedByOthers := func(obj *unstructured.Unstructured) {
		_ = unstructured.SetNestedField(obj.Object, int64(5), "spec", "replicas")
		unstructured.RemoveNestedField(obj.Object, "metadata", "labels", "tier")
		_ = unstructured.SetNestedStringSlice(obj.Object, []string{"theirs"}, "metadata", "finalizers")
		containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
		containers[1].(map[string]any)["image"] = "nginx:2"
		containers[1].(map[string]any)["args"] = []any{"--a", "--b"}
		_ = unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers")
	}
	// asWritten stores the cpu request with the spelling of the manifest.
	asWritten := func(obj *unstructured.Unstructured) {
		containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
		containers[1].(map[string]any)["resources"] = map[string]any{"requests": map[string]any{"cpu": 0.5, "memory": "1024Mi"}}
		_ = unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers")
	}
	changed := []string{
		`metadata.finalizers["keep"]`, "metadata.labels.tier", "spec.replicas",
		"spec.template.spec.containers[name=web].args", "spec.template.spec.containers[name=web].image",
	}
	for name, tc := range map[string]struct {
		live     *unstructured.Unstructured
		accepted bool
		computed []string
		want     []string
	}{
		// Server defaults, the server's spelling of quantities and what other
		// clients added are not drift.
		"in sync": {live: liveObject(t, nil, entry("Terraform", "Apply", testFields), entry("other", "Update", testOtherFields))},
		"changed by other clients": {
			live: liveObject(t, changedByOthers, entry("Terraform", "Apply", testFieldsAfterChanges), entry("other", "Update", testOtherFields)),
			want: changed,
		},
		// Without an entry of its own, as after a state move, values are compared.
		"never applied, same values": {
			live: liveObject(t, asWritten, entry("kubectl-client-side-apply", "Update", testFields), entry("other", "Update", testOtherFields)),
		},
		// A number where the cluster stores a string is a difference until
		// the first apply: without the schema it cannot be told from a change.
		"never applied, another spelling": {
			live: liveObject(t, nil, entry("kubectl-client-side-apply", "Update", testFields), entry("other", "Update", testOtherFields)),
			want: []string{"spec.template.spec.containers[name=web].resources.requests.cpu"},
		},
		"never applied, changed": {
			live: liveObject(t, func(obj *unstructured.Unstructured) { asWritten(obj); changedByOthers(obj) }, entry("kubectl-client-side-apply", "Update", testFieldsAfterChanges), entry("other", "Update", testOtherFields)),
			want: changed,
		},
		"computed fields are not compared": {
			live:     liveObject(t, changedByOthers, entry("Terraform", "Apply", testFieldsAfterChanges), entry("other", "Update", testOtherFields)),
			computed: []string{"spec.replicas", "metadata.labels", "spec.template.spec.containers[0]", `metadata.finalizers`},
		},
		"differences accepted at apply": {
			live:     liveObject(t, changedByOthers, entry("Terraform", "Apply", testFieldsAfterChanges), entry("other", "Update", testOtherFields)),
			accepted: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			manifest := decodeJSON(t, testManifest)
			var accepted map[string]string
			if tc.accepted {
				_, accepted = liveChanges(manifest, tc.live, "Terraform", nil, nil)
			}
			result, found := liveChanges(manifest, tc.live, "Terraform", tc.computed, accepted)
			var paths []string
			for path := range found {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			if !reflect.DeepEqual(paths, tc.want) {
				t.Fatalf("changed fields:\n got %q\nwant %q", paths, tc.want)
			}
			if len(tc.want) == 0 {
				if !reflect.DeepEqual(result, manifest) {
					t.Fatalf("manifest must stay as written, got %v", result)
				}
				return
			}
			if !reflect.DeepEqual(tc.want, changed) {
				return
			}
			container := result["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)[0].(map[string]any)
			labels := result["metadata"].(map[string]any)["labels"]
			got := []any{result["spec"].(map[string]any)["replicas"], labels, container["image"], container["args"], container["resources"], result["metadata"].(map[string]any)["finalizers"]}
			want := []any{int64(5), map[string]any{"app": "web"}, "nginx:2", []any{"--a", "--b"}, map[string]any{"requests": map[string]any{"cpu": 0.5, "memory": "1024Mi"}}, []any{}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("manifest with live changes:\n got %v\nwant %v", got, want)
			}
		})
	}
}

// An apply leaves out a computed field another client has taken, so that
// Terraform does not become an owner of it again, and sends everything else
// as configured.
func TestLeavingComputed(t *testing.T) {
	live := liveObject(t, func(obj *unstructured.Unstructured) {
		_ = unstructured.SetNestedField(obj.Object, int64(5), "spec", "replicas")
		unstructured.RemoveNestedField(obj.Object, "metadata", "labels", "tier")
	}, entry("Terraform", "Apply", testFieldsAfterChanges), entry("other", "Update", testOtherFields))
	manifest := decodeJSON(t, testManifest)
	sent := leavingComputed(manifest, live, "Terraform", []string{"spec.replicas", "spec.selector"})
	if _, present := sent["spec"].(map[string]any)["replicas"]; present {
		t.Error("replicas, which the other client owns, must not be sent")
	}
	if sent["spec"].(map[string]any)["selector"] == nil {
		t.Error("a computed field this manager still owns must be sent")
	}
	if got := sent["metadata"].(map[string]any)["labels"].(map[string]any)["tier"]; got != "front" {
		t.Errorf("a field that is not computed must be sent as configured, got %v", got)
	}
	if !reflect.DeepEqual(leavingComputed(manifest, live, "Terraform", nil), manifest) {
		t.Error("without computed fields the manifest is sent as it is")
	}
}

// An imported object that this resource applied before reads as the manifest
// it was applied from, in the server's spelling.
func TestExportedOwnedFields(t *testing.T) {
	live := liveObject(t, nil, entry("Terraform", "Apply", testFields), entry("other", "Update", testOtherFields))
	want := decodeJSON(t, testManifest)
	want["metadata"].(map[string]any)["namespace"] = "default"
	podSpec := want["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)
	delete(podSpec, "tolerations")
	delete(podSpec, "nodeSelector")
	podSpec["containers"].([]any)[0].(map[string]any)["resources"] = map[string]any{"requests": map[string]any{"cpu": "500m", "memory": "1024Mi"}}
	if got := exported(live, "Terraform"); !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got %v\nwant %v", got, want)
	}
	if got := exported(live, "someone-else"); got["spec"].(map[string]any)["revisionHistoryLimit"] != int64(10) || got["status"] != nil {
		t.Fatalf("an object applied by others must be exported in full, without status: %v", got)
	}
}

// Values are compared as JSON: types matter, and numbers compare by value.
func TestSame(t *testing.T) {
	for _, tc := range []struct {
		want, have any
		same       bool
	}{
		{"a", "a", true}, {true, true, true}, {int64(1), float64(1), true}, {0.5, 0.5, true},
		{true, "true", false}, {int64(1), "1", false}, {0.5, "500m", false}, {"01", "1", false}, {"1024Mi", "1Gi", false}, {int64(2), int64(1), false},
	} {
		if got := same(tc.want, tc.have); got != tc.same {
			t.Errorf("same(%v, %v) = %t", tc.want, tc.have, got)
		}
	}
}
