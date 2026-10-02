// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestFlattenDeploymentSpecForFramework_PausedFalsePreserved(t *testing.T) {
	spec := appsv1.DeploymentSpec{Paused: false}
	flattened, err := FlattenDeploymentSpecForFramework(spec)
	if err != nil {
		t.Fatalf("unexpected flatten error: %v", err)
	}
	if len(flattened) != 1 {
		t.Fatalf("expected one spec entry, got %d", len(flattened))
	}
	entry := flattened[0].(map[string]interface{})
	paused, ok := entry["paused"]
	if !ok {
		t.Fatalf("expected paused field to be present in flattened deployment spec")
	}
	if paused.(bool) {
		t.Fatalf("expected paused=false, got true")
	}
}

func TestFlattenDeploymentSpecForFramework_PausedTruePreserved(t *testing.T) {
	spec := appsv1.DeploymentSpec{Paused: true}
	flattened, err := FlattenDeploymentSpecForFramework(spec)
	if err != nil {
		t.Fatalf("unexpected flatten error: %v", err)
	}
	entry := flattened[0].(map[string]interface{})
	paused, ok := entry["paused"]
	if !ok {
		t.Fatalf("expected paused field to be present in flattened deployment spec")
	}
	if !paused.(bool) {
		t.Fatalf("expected paused=true, got false")
	}
}
