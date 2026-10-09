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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestUpgradeState(t *testing.T) {
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
				err := UpgradeState(state, "state", []string{"spec"}, tc.legacy)
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
				if err := UpgradeState(state, "state", []string{"spec"}, tc.legacy); err != nil {
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
	for _, input := range []string{`{"spec":[{},{}]}`, `{"spec":[null]}`, `{"spec":{}}`, `{"spec":[{"container":[null]}]}`, `{"spec":[{"volume":[{"image":[{},{}]}]}]}`, `{"spec":[{"volume":[{"image":[null]}]}]}`} {
		var state map[string]any
		_ = json.Unmarshal([]byte(input), &state)
		if err := UpgradeState(state, "state", []string{"spec"}, false); err == nil {
			t.Fatalf("accepted malformed state %s", input)
		}
	}
}

func TestPodFeatureStateMigration(t *testing.T) {
	ctx := context.Background()
	for owner, option := range map[string]Options{"deployment": Deployment(), "daemonset": DaemonSet(), "statefulset": StatefulSet(), "pod": Pod(), "job": Job(), "cronjob": CronJob()} {
		for _, populated := range []bool{false, true} {
			t.Run(owner+"/"+map[bool]string{false: "absent", true: "configured"}[populated], func(t *testing.T) {
				b := For(option)
				raw := `{"spec":[{"container":[{"name":"app","security_context":[{}]}],"init_container":[{"name":"init","security_context":[{}]}],"volume":[{"name":"data","empty_dir":[{}]}]}]}`
				if populated {
					raw = `{"spec":[{"host_users":false,"container":[{"name":"app","security_context":[{"proc_mount":"Unmasked"}]}],"init_container":[{"name":"init","security_context":[{"proc_mount":"Unmasked"}]}],"volume":[{"name":"data","image":[{"reference":"registry.example/data:v1","pull_policy":"IfNotPresent"}]}]}]}`
				}
				var object map[string]any
				if err := json.Unmarshal([]byte(raw), &object); err != nil {
					t.Fatal(err)
				}
				if err := UpgradeState(object, "state", []string{"spec"}, false); err != nil {
					t.Fatal(err)
				}
				data, _ := json.Marshal(object)
				if err := UpgradeState(object, "state", []string{"spec"}, false); err != nil {
					t.Fatal(err)
				}
				again, _ := json.Marshal(object)
				if string(data) != string(again) {
					t.Fatal("upgrade was not idempotent")
				}
				s := schema.Schema{Blocks: map[string]schema.Block{"spec": b.Spec}}
				decoded, err := (&tfprotov6.RawState{JSON: data}).Unmarshal(s.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: s, Raw: decoded}
				var baseline types.List
				if d := state.GetAttribute(ctx, path.Root("spec"), &baseline); d.HasError() {
					t.Fatal(d)
				}
				live, d := b.ExpandSpec(ctx, baseline, nil, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				if populated {
					if live.HostUsers == nil || *live.HostUsers || live.Containers[0].SecurityContext.ProcMount == nil || *live.Containers[0].SecurityContext.ProcMount != corev1.UnmaskedProcMount || live.InitContainers[0].SecurityContext.ProcMount == nil || live.Volumes[0].Image == nil || live.Volumes[0].Image.Reference != "registry.example/data:v1" {
						t.Fatalf("configured features lost: %#v", live)
					}
				} else if live.HostUsers != nil || live.Containers[0].SecurityContext.ProcMount != nil || live.InitContainers[0].SecurityContext.ProcMount != nil || live.Volumes[0].Image != nil {
					t.Fatalf("legacy omission changed API payload: %#v", live)
				}
				refreshed, d := b.RefreshSpec(ctx, live, baseline, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				current := tfsdk.State{Schema: s, Raw: decoded}
				if d := current.SetAttribute(ctx, path.Root("spec"), refreshed); d.HasError() {
					t.Fatal(d)
				}
				at := path.Root("spec").AtListIndex(0)
				paths := []path.Path{at.AtName("host_users"), at.AtName("volume").AtListIndex(0).AtName("image")}
				for _, kind := range []string{"container", "init_container"} {
					paths = append(paths, at.AtName(kind).AtListIndex(0).AtName("security_context").AtListIndex(0).AtName("proc_mount"))
				}
				for _, at := range paths {
					var before, after attr.Value
					if d := state.GetAttribute(ctx, at, &before); d.HasError() {
						t.Fatal(d)
					}
					if d := current.GetAttribute(ctx, at, &after); d.HasError() {
						t.Fatal(d)
					}
					if !before.Equal(after) || (!populated && !after.IsNull()) {
						t.Fatalf("%s: upgraded %s, refreshed %s", at, before, after)
					}
				}
			})
		}
	}
}

func TestAbsentPodFeatureDefaults(t *testing.T) {
	ctx := context.Background()
	b := For(Pod())
	live := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", SecurityContext: &corev1.SecurityContext{}}}}
	baseline, d := b.RefreshSpec(ctx, live, types.ListNull(b.ObjectType()), path.Root("spec"))
	if d.HasError() {
		t.Fatal(d)
	}
	// Explicit false and Unmasked remain observable when a later API response
	// drops the settings; effective defaults must not be mistaken for isolation.
	live.HostUsers = ptr.To(false)
	live.Containers[0].SecurityContext.ProcMount = ptr.To(corev1.UnmaskedProcMount)
	configured, d := b.RefreshSpec(ctx, live, baseline, path.Root("spec"))
	if d.HasError() {
		t.Fatal(d)
	}
	live.HostUsers = nil
	live.Containers[0].SecurityContext.ProcMount = nil
	refreshed, d := b.RefreshSpec(ctx, live, configured, path.Root("spec"))
	if d.HasError() {
		t.Fatal(d)
	}
	values := refreshed.Elements()[0].(types.Object).Attributes()
	if !values["host_users"].Equal(types.BoolValue(true)) {
		t.Fatalf("dropped host_users=false hidden: %s", values["host_users"])
	}
	sc := values["container"].(types.List).Elements()[0].(types.Object).Attributes()["security_context"].(types.List).Elements()[0].(types.Object).Attributes()
	if !sc["proc_mount"].Equal(types.StringValue("Default")) {
		t.Fatalf("dropped Unmasked hidden: %s", sc["proc_mount"])
	}
}

func TestImageVolumeOwnershipAcrossRefresh(t *testing.T) {
	ctx := context.Background()
	for owner, option := range map[string]Options{"deployment": Deployment(), "daemonset": DaemonSet(), "statefulset": StatefulSet(), "pod": Pod(), "job": Job(), "cronjob": CronJob()} {
		t.Run(owner, func(t *testing.T) {
			b := For(option)
			live := corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}, Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{Image: &corev1.ImageVolumeSource{Reference: "data:v1", PullPolicy: corev1.PullIfNotPresent}}},
				{Name: "other", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			}}
			imported, d := b.RefreshSpec(ctx, live, types.ListNull(b.ObjectType()), path.Root("spec"))
			if d.HasError() {
				t.Fatal(d)
			}
			image := get(imported, 0, "volume", 0, "image").(types.Object)
			if image.IsNull() || !image.Attributes()["reference"].Equal(types.StringValue("data:v1")) {
				t.Fatal("import must discover the existing image")
			}
			legacy := with(t, imported, types.ObjectNull(image.AttributeTypes(ctx)), 0, "volume", 0, "image").(types.List)
			for name, flatten := range map[string]func(context.Context, corev1.PodSpec, types.List, path.Path) (types.List, diag.Diagnostics){"refresh": b.RefreshSpec, "write": b.FlattenSpec} {
				t.Run(name, func(t *testing.T) {
					unconfigured, d := flatten(ctx, live, legacy, path.Root("spec"))
					if d.HasError() || !get(unconfigured, 0, "volume", 0, "image").IsNull() {
						t.Fatalf("unconfigured admission image adopted: %s; %v", unconfigured, d)
					}
					changed := live.DeepCopy()
					changed.Volumes[0].Image.Reference = "data:v2"
					observed, d := flatten(ctx, *changed, imported, path.Root("spec"))
					if d.HasError() || get(observed, 0, "volume", 0, "image").IsNull() || !get(observed, 0, "volume", 0, "image", "reference").Equal(types.StringValue("data:v2")) {
						t.Fatalf("configured image drift hidden: %s; %v", observed, d)
					}
					changed.Volumes[0], changed.Volumes[1] = changed.Volumes[1], changed.Volumes[0]
					observed, d = flatten(ctx, *changed, imported, path.Root("spec"))
					if d.HasError() || get(observed, 0, "volume", 1, "image").IsNull() || !get(observed, 0, "volume", 1, "image", "reference").Equal(types.StringValue("data:v2")) {
						t.Fatalf("reordering hid the configured image: %s; %v", observed, d)
					}
				})
			}
		})
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
				if err := UpgradeState(values, "state", []string{"spec"}, false); err != nil {
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
			// Resources is an object, not a list probe; block presence rules are unchanged.
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
