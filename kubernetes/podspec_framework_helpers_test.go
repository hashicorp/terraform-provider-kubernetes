// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"encoding/json"
	"testing"
)

func TestFrameworkResourcesAdapterPreservesInput(t *testing.T) {
	container := map[string]interface{}{"name": "app", "resources": map[string]interface{}{"limits": map[string]interface{}{"cpu": "500m"}, "requests": map[string]interface{}{}}}
	spec := []interface{}{map[string]interface{}{"container": []interface{}{container}, "init_container": []interface{}{container}}}
	before, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := ExpandPodSpecForFramework(spec)
	if err != nil {
		t.Fatal(err)
	}
	if expanded.Containers[0].Resources.Limits.Cpu().String() != "500m" || expanded.InitContainers[0].Resources.Limits.Cpu().String() != "500m" {
		t.Fatal("resources omitted by SDK adapter")
	}
	after, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("adapter mutated caller input")
	}
	flat, err := FlattenPodSpecForFramework(*expanded)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"container", "init_container"} {
		got := flat[0].(map[string]interface{})[kind].([]interface{})[0].(map[string]interface{})["resources"]
		if _, ok := got.(map[string]interface{}); !ok {
			t.Fatalf("%s.resources remains %T", kind, got)
		}
	}
}
