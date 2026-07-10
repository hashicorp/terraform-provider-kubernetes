// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestGenericEphemeralVolumeClaimTemplateForceNew(t *testing.T) {
	tests := []struct {
		name          string
		root          map[string]*schema.Schema
		claimSpecPath []string
		wantForceNew  bool
	}{
		{
			name:          "deployment pod template",
			root:          resourceKubernetesDeploymentSchemaV1(),
			claimSpecPath: []string{"spec", "template", "spec", "volume", "ephemeral", "volume_claim_template", "spec"},
			wantForceNew:  false,
		},
		{
			name:          "standalone pod",
			root:          resourceKubernetesPodSchemaV1(),
			claimSpecPath: []string{"spec", "volume", "ephemeral", "volume_claim_template", "spec"},
			wantForceNew:  true,
		},
		{
			name:          "standalone persistent volume claim",
			root:          persistentVolumeClaimFields(),
			claimSpecPath: []string{"spec"},
			wantForceNew:  true,
		},
	}

	forceNewPaths := [][]string{
		{"access_modes"},
		{"resources", "limits"},
		{"selector"},
		{"selector", "match_expressions"},
		{"selector", "match_expressions", "key"},
		{"selector", "match_expressions", "operator"},
		{"selector", "match_expressions", "values"},
		{"selector", "match_labels"},
		{"volume_name"},
		{"storage_class_name"},
		{"volume_mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claimSpec := nestedResourceSchema(t, tt.root, tt.claimSpecPath...)

			for _, path := range forceNewPaths {
				field := nestedSchemaField(t, claimSpec, path...)
				if got := field.ForceNew; got != tt.wantForceNew {
					t.Errorf("ForceNew at %v = %t, want %t", path, got, tt.wantForceNew)
				}
			}

			if requests := nestedSchemaField(t, claimSpec, "resources", "requests"); requests.ForceNew {
				t.Error("resources.requests must remain updateable for persistent volume claim expansion")
			}
		})
	}
}

func nestedResourceSchema(t *testing.T, root map[string]*schema.Schema, path ...string) map[string]*schema.Schema {
	t.Helper()

	for _, name := range path {
		field, ok := root[name]
		if !ok {
			t.Fatalf("schema field %q not found in path %v", name, path)
		}
		resource, ok := field.Elem.(*schema.Resource)
		if !ok {
			t.Fatalf("schema field %q in path %v does not contain a nested resource", name, path)
		}
		root = resource.Schema
	}

	return root
}

func nestedSchemaField(t *testing.T, root map[string]*schema.Schema, path ...string) *schema.Schema {
	t.Helper()

	for i, name := range path {
		field, ok := root[name]
		if !ok {
			t.Fatalf("schema field %q not found in path %v", name, path)
		}
		if i == len(path)-1 {
			return field
		}
		resource, ok := field.Elem.(*schema.Resource)
		if !ok {
			t.Fatalf("schema field %q in path %v does not contain a nested resource", name, path)
		}
		root = resource.Schema
	}

	t.Fatal("schema field path must not be empty")
	return nil
}
