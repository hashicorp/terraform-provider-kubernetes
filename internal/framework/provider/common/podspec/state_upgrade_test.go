// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestUpgradeResourcesState(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		legacy            bool
		errorPath         string
	}{
		{name: "absent", input: `{}`, want: `{}`},
		{name: "null", input: `{"resources":null}`, want: `{"resources":null}`},
		{name: "empty list", input: `{"resources":[]}`, want: `{"resources":{"limits":{},"requests":{}}}`},
		{name: "empty element", input: `{"resources":[{}]}`, want: `{"resources":{"limits":{},"requests":{}}}`},
		{name: "object", input: `{"resources":{"limits":{},"requests":null}}`, want: `{"resources":{"limits":{},"requests":null}}`},
		{name: "populated", input: `{"resources":[{"limits":{"cpu":"0.5","example.com/device":"2"},"requests":null}]}`, want: `{"resources":{"limits":{"cpu":"0.5","example.com/device":"2"},"requests":null}}`},
		{name: "legacy quantities", input: `{"resources":[{"limits":[{"cpu":"0.5"}],"requests":[]}]}`, want: `{"resources":{"limits":{"cpu":"0.5"},"requests":{}}}`, legacy: true},
		{name: "legacy absent quantity", input: `{"resources":[{"limits":null}]}`, want: `{"resources":{"limits":{},"requests":{}}}`, legacy: true},
		{name: "multiple resources", input: `{"resources":[{},{}]}`, errorPath: "resources"},
		{name: "null resources element", input: `{"resources":[null]}`, errorPath: "resources"},
		{name: "scalar", input: `{"resources":1}`, errorPath: "resources"},
		{name: "multiple quantities", input: `{"resources":[{"limits":[{},{}]}]}`, legacy: true, errorPath: "resources.limits"},
		{name: "invalid quantity element", input: `{"resources":[{"limits":[null]}]}`, legacy: true, errorPath: "resources.limits[0]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"container", "init_container"} {
				var container, want map[string]any
				if err := json.Unmarshal([]byte(tc.input), &container); err != nil {
					t.Fatal(err)
				}
				container["name"] = "app"
				sibling := []any{map[string]any{"name": "data"}}
				state := map[string]any{"spec": []any{map[string]any{kind: []any{container}, "volume": sibling}}}
				err := UpgradeResourcesState(state, "state", []string{"spec"}, tc.legacy)
				if tc.errorPath != "" {
					if err == nil || !strings.Contains(err.Error(), "state.spec[0]."+kind+"[0]."+tc.errorPath) {
						t.Fatalf("error = %v", err)
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatal(err)
				}
				want["name"] = "app"
				if !reflect.DeepEqual(container, want) {
					t.Fatalf("got %#v, want %#v", container, want)
				}
				before, _ := json.Marshal(state)
				if err := UpgradeResourcesState(state, "state", []string{"spec"}, tc.legacy); err != nil {
					t.Fatal(err)
				}
				after, _ := json.Marshal(state)
				if string(before) != string(after) {
					t.Fatal("conversion is not idempotent")
				}
				if !reflect.DeepEqual(state["spec"].([]any)[0].(map[string]any)["volume"], sibling) {
					t.Fatal("sibling changed")
				}
			}
		})
	}
	for _, input := range []string{`{"spec":[{},{}]}`, `{"spec":[null]}`, `{"spec":{}}`, `{"spec":[{"container":[null]}]}`} {
		var state map[string]any
		_ = json.Unmarshal([]byte(input), &state)
		if err := UpgradeResourcesState(state, "state", []string{"spec"}, false); err == nil {
			t.Fatalf("accepted malformed state %s", input)
		}
	}
}

// Compare the upgraded representation with an actual API flatten for every
// consumer and both container kinds, not just Terraform's planned actions.
func TestResourcesUpgradeMatchesRefresh(t *testing.T) {
	ctx := context.Background()
	for name, option := range map[string]Options{"deployment": Deployment(), "daemonset": DaemonSet(), "statefulset": StatefulSet(), "pod": Pod(), "job": Job(), "cronjob": CronJob()} {
		t.Run(name, func(t *testing.T) {
			b := For(option)
			for _, old := range []string{`[]`, `[{}]`, `[{"limits":{},"requests":{}}]`, `[{"limits":{"cpu":"500m"},"requests":{}}]`} {
				var values map[string]any
				raw := `{"spec":[{"container":[{"name":"app","resources":` + old + `}],"init_container":[{"name":"init","resources":` + old + `}]}]}`
				if err := json.Unmarshal([]byte(raw), &values); err != nil {
					t.Fatal(err)
				}
				if err := UpgradeResourcesState(values, "state", []string{"spec"}, false); err != nil {
					t.Fatal(err)
				}
				data, _ := json.Marshal(values)
				s := schema.Schema{Blocks: map[string]schema.Block{"spec": b.Spec}}
				value, err := (&tfprotov6.RawState{JSON: data}).Unmarshal(s.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: s, Raw: value}
				var baseline types.List
				if d := state.GetAttribute(ctx, path.Root("spec"), &baseline); d.HasError() {
					t.Fatal(d)
				}
				live := corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}, InitContainers: []corev1.Container{{Name: "init"}}}
				if strings.Contains(old, "500m") {
					live.Containers[0].Resources.Limits = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")}
					live.InitContainers[0].Resources = live.Containers[0].Resources
				}
				refreshed, d := b.RefreshSpec(ctx, live, baseline, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				selectResources := func(v types.List, kind string) attr.Value {
					return v.Elements()[0].(types.Object).Attributes()[kind].(types.List).Elements()[0].(types.Object).Attributes()["resources"]
				}
				for _, kind := range []string{"container", "init_container"} {
					if got, want := selectResources(refreshed, kind), selectResources(baseline, kind); !got.Equal(want) {
						t.Fatalf("%s %s: refresh %s != upgrade %s", kind, old, got, want)
					}
				}
				expanded, d := b.ExpandSpec(ctx, baseline, nil, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				if !reflect.DeepEqual(expanded.Containers[0].Resources, live.Containers[0].Resources) || !reflect.DeepEqual(expanded.InitContainers[0].Resources, live.InitContainers[0].Resources) {
					t.Fatalf("resources lost during expansion: %#v", expanded)
				}
			}
			// Resources is no longer a list probe; unrelated presence policies remain.
			for _, key := range []string{"spec.container.resources", "spec.init_container.resources"} {
				if b.absentZero[key] || b.zeroAbsent[key] {
					t.Fatalf("object retained as list probe: %s", key)
				}
			}
			if b.zeroAbsent["spec.container.security_context"] || !b.zeroAbsent["spec.security_context"] || !b.zeroAbsent["spec.dns_config"] {
				t.Fatal("unrelated block presence policy changed")
			}
		})
	}
}
