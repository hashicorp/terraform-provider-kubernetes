// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestWorkloadSelectorReadPreservesEmpty(t *testing.T) {
	for _, workload := range []struct {
		name     string
		resource resource.Resource
		claim    bool
	}{
		{"deployment", &DeploymentV1{}, false},
		{"daemon-set", &DaemonSetV1{}, false},
		{"stateful-set", &StatefulSetV1{}, false},
		{"stateful-set-claim", &StatefulSetV1{}, true},
	} {
		for name, selector := range map[string]string{
			"empty-leaves":      `{"match_labels":{},"match_expressions":[{"key":"app","operator":"Exists","values":[]}]}`,
			"empty-expressions": `{"match_labels":{"app":"database"},"match_expressions":[]}`,
		} {
			t.Run(workload.name+"/"+name, func(t *testing.T) {
				ctx := context.Background()
				state := workloadSelectorState(t, workload.resource, selector, workload.claim)
				before := state.Raw.Copy()
				switch workload.resource.(type) {
				case *DeploymentV1:
					var baseline DeploymentV1Model
					if d := state.Get(ctx, &baseline); d.HasError() {
						t.Fatal(d)
					}
					spec, d := expandDeploymentSpec(ctx, baseline.Spec, path.Root("spec"))
					if d.HasError() {
						t.Fatal(d)
					}
					api := workloadSelectorAPIRoundTrip(t, spec)
					value, d := flattenDeploymentSpec(ctx, *api, baseline.Spec, path.Root("spec"))
					if d.HasError() {
						t.Fatal(d)
					}
					if d := state.SetAttribute(ctx, path.Root("spec"), value); d.HasError() {
						t.Fatal(d)
					}
				case *DaemonSetV1:
					var baseline DaemonSetV1Model
					if d := state.Get(ctx, &baseline); d.HasError() {
						t.Fatal(d)
					}
					spec, d := expandDaemonSetSpecModel(ctx, baseline.Spec, path.Root("spec"))
					if d.HasError() {
						t.Fatal(d)
					}
					api := workloadSelectorAPIRoundTrip(t, spec)
					value, d := flattenDaemonSetSpecModel(ctx, api, baseline.Spec)
					if d.HasError() {
						t.Fatal(d)
					}
					if d := state.SetAttribute(ctx, path.Root("spec"), value); d.HasError() {
						t.Fatal(d)
					}
				case *StatefulSetV1:
					var baseline StatefulSetV1Model
					if d := state.Get(ctx, &baseline); d.HasError() {
						t.Fatal(d)
					}
					spec, d := expandStatefulSetSpec(ctx, baseline.Spec[0])
					if d.HasError() {
						t.Fatal(d)
					}
					api := workloadSelectorAPIRoundTrip(t, spec)
					value, d := flattenStatefulSetSpec(ctx, *api, &baseline.Spec[0], testMetadataFilters{})
					if d.HasError() {
						t.Fatal(d)
					}
					if d := state.SetAttribute(ctx, path.Root("spec"), []StatefulSetSpecModel{value}); d.HasError() {
						t.Fatal(d)
					}
				}
				at := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0)
				if workload.claim {
					at = at.WithAttributeName("volume_claim_template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0)
				}
				at = at.WithAttributeName("selector")
				prior, _, err := tftypes.WalkAttributePath(before, at)
				if err != nil {
					t.Fatal(err)
				}
				current, _, err := tftypes.WalkAttributePath(state.Raw, at)
				if err != nil {
					t.Fatal(err)
				}
				if !prior.(tftypes.Value).Equal(current.(tftypes.Value)) {
					t.Fatalf("Read changed configured selector: before %s; after %s", prior, current)
				}
			})
		}
	}
}

func TestWorkloadSelectorNormalizationNoPatch(t *testing.T) {
	for _, workload := range []struct {
		name     string
		resource resource.Resource
		claim    bool
	}{
		{"deployment", &DeploymentV1{}, false},
		{"daemon-set", &DaemonSetV1{}, false},
		{"stateful-set", &StatefulSetV1{}, false},
		{"stateful-set-claim", &StatefulSetV1{}, true},
	} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%t", workload.name, reverse), func(t *testing.T) {
				ctx := context.Background()
				before, after := `{"match_labels":{},"match_expressions":[{"key":"app","operator":"Exists","values":[]}]}`, `{"match_expressions":[{"key":"app","operator":"Exists"}]}`
				if reverse {
					before, after = after, before
				}
				state := workloadSelectorState(t, workload.resource, before, workload.claim)
				plan := workloadSelectorState(t, workload.resource, after, workload.claim)
				switch workload.resource.(type) {
				case *DeploymentV1:
					var old, next DeploymentV1Model
					if d := state.Get(ctx, &old); d.HasError() {
						t.Fatal(d)
					}
					if d := plan.Get(ctx, &next); d.HasError() {
						t.Fatal(d)
					}
					original, d := expandDeploymentSpec(ctx, old.Spec, path.Root("spec"))
					if d.HasError() {
						t.Fatal(d)
					}
					modified, d := expandDeploymentSpec(ctx, next.Spec, path.Root("spec"))
					if d.HasError() {
						t.Fatal(d)
					}
					patch, err := deploymentSpecPatch(*original, *modified, *original)
					if err != nil || string(patch) != "{}" {
						t.Fatalf("normalization patch = %s; error: %v", patch, err)
					}
				case *DaemonSetV1:
					var old, next DaemonSetV1Model
					if d := state.Get(ctx, &old); d.HasError() {
						t.Fatal(d)
					}
					if d := plan.Get(ctx, &next); d.HasError() {
						t.Fatal(d)
					}
					patch, d := daemonSetStrategicSpecPatch(ctx, old.Spec, next.Spec, nil)
					if d.HasError() || len(patch) != 0 {
						t.Fatalf("normalization patch = %s; diagnostics: %v", patch, d)
					}
				case *StatefulSetV1:
					var old, next StatefulSetV1Model
					if d := state.Get(ctx, &old); d.HasError() {
						t.Fatal(d)
					}
					if d := plan.Get(ctx, &next); d.HasError() {
						t.Fatal(d)
					}
					original, d := expandStatefulSetSpec(ctx, old.Spec[0])
					if d.HasError() {
						t.Fatal(d)
					}
					patch, d := (&StatefulSetV1{}).patchStatefulSetSpec(ctx, next.Spec[0], old.Spec[0], *original)
					if d.HasError() || len(patch) != 0 {
						t.Fatalf("normalization patch = %+v; diagnostics: %v", patch, d)
					}
				}
			})
		}
	}
}

func workloadSelectorState(t *testing.T, r resource.Resource, selector string, claim bool) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	response := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	extra := ""
	if _, ok := r.(*StatefulSetV1); ok {
		extra = `"service_name":"database",`
	}
	if claim {
		extra += fmt.Sprintf(`"volume_claim_template":[{
		  "metadata":[{"name":"data","namespace":"default"}],
		  "spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"selector":[%s]}]
		}],`, selector)
		selector = `{"match_labels":{"app":"database"}}`
	}
	raw := fmt.Sprintf(`{
	  "id":"default/selector-regression",
	  "metadata":[{"name":"selector-regression","namespace":"default"}],
	  "spec":[{%s"selector":[%s],
	    "template":[{"metadata":[{"labels":{"app":"database"}}],
	      "spec":[{"container":[{"name":"main","image":"busybox:1.36"}]}]}]
	  }]
	}`, extra, selector)
	value, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(response.Schema.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: response.Schema, Raw: value}
}

func workloadSelectorAPIRoundTrip[T any](t *testing.T, value T) T {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result T
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
