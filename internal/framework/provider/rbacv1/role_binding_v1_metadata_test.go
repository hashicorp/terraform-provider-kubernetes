// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleBindingMetadataPatchPreservesExternalKeys(t *testing.T) {
	ctx := context.Background()
	for _, field := range []string{"annotations", "labels"} {
		for _, existing := range []map[string]string{nil, {"external.example.com/note": "keep"}} {
			t.Run(fmt.Sprintf("%s/existing=%d", field, len(existing)), func(t *testing.T) {
				state := common.MetadataModel{MetadataBase: common.MetadataBase{
					Annotations: types.MapNull(types.StringType), Labels: types.MapNull(types.StringType),
				}}
				plan := state
				managed, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"managed.example.com/key": "new"})
				if diags.HasError() {
					t.Fatal(diags)
				}
				current := metav1.ObjectMeta{}
				if field == "annotations" {
					plan.Annotations = managed
					current.Annotations = existing
				} else {
					plan.Labels = managed
					current.Labels = existing
				}
				before, err := json.Marshal(map[string]any{"metadata": current})
				if err != nil {
					t.Fatal(err)
				}
				patchJSON, err := json.Marshal(roleBindingMetadataPatchOps(state, plan, current))
				if err != nil {
					t.Fatal(err)
				}
				patch, err := jsonpatch.DecodePatch(patchJSON)
				if err != nil {
					t.Fatal(err)
				}
				after, err := patch.Apply(before)
				if err != nil {
					t.Fatal(err)
				}
				var got struct {
					Metadata metav1.ObjectMeta `json:"metadata"`
				}
				if err := json.Unmarshal(after, &got); err != nil {
					t.Fatal(err)
				}
				values := got.Metadata.Annotations
				if field == "labels" {
					values = got.Metadata.Labels
				}
				if values["managed.example.com/key"] != "new" {
					t.Fatalf("managed key missing: %s", after)
				}
				for key, value := range existing {
					if values[key] != value {
						t.Fatalf("external key lost: %s", after)
					}
				}

				// Removing the managed key must leave external ownership intact.
				patchJSON, err = json.Marshal(roleBindingMetadataPatchOps(plan, state, got.Metadata))
				if err != nil {
					t.Fatal(err)
				}
				patch, err = jsonpatch.DecodePatch(patchJSON)
				if err != nil {
					t.Fatal(err)
				}
				removed, err := patch.Apply(after)
				if err != nil {
					t.Fatal(err)
				}
				// Decode into a fresh object to avoid retaining removed map entries.
				var result map[string]map[string]map[string]string
				if err := json.Unmarshal(removed, &result); err != nil {
					t.Fatal(err)
				}
				if _, exists := result["metadata"][field]["managed.example.com/key"]; exists {
					t.Fatalf("managed key retained: %s", removed)
				}
				for key, value := range existing {
					if result["metadata"][field][key] != value {
						t.Fatalf("external key lost on removal: %s", removed)
					}
				}
			})
		}
	}
}
