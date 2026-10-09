// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				paths := []path.Path{at.AtName("host_users"), at.AtName("set_hostname_as_fqdn"), at.AtName("volume").AtListIndex(0).AtName("image")}
				for _, kind := range []string{"container", "init_container"} {
					for _, name := range []string{"proc_mount", "app_armor_profile", "windows_options"} {
						paths = append(paths, at.AtName(kind).AtListIndex(0).AtName("security_context").AtListIndex(0).AtName(name))
					}
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

func TestOptionalPodFeaturesRoundTrip(t *testing.T) {
	ctx := context.Background()
	for _, option := range []Options{Pod(), Deployment(), DaemonSet(), StatefulSet(), Job(), CronJob()} {
		b := For(option)
		for name, raw := range map[string]string{
			"linux":   `{"spec":[{"set_hostname_as_fqdn":true,"security_context":[{"supplemental_groups_policy":"Strict","se_linux_change_policy":"Recursive","app_armor_profile":{"type":"RuntimeDefault"}}],"container":[{"name":"app","security_context":[{"app_armor_profile":{"type":"Localhost","localhost_profile":"example"}}],"liveness_probe":[{"tcp_socket":[{"host":"127.0.0.1","port":"8080"}],"termination_grace_period_seconds":5}],"startup_probe":[{"termination_grace_period_seconds":7}],"lifecycle":[{"pre_stop":[{"sleep":{"seconds":0}}]}],"volume_mount":[{"name":"data","mount_path":"/data","read_only":true,"recursive_read_only":"Enabled"}],"port":[{"container_port":5000,"protocol":"SCTP"}]}]}]}`,
			"empty":   `{"spec":[{"container":[{"name":"app","liveness_probe":[{"tcp_socket":[{"port":"80","host":""}]}]}],"affinity":[{"pod_affinity":[{"required_during_scheduling_ignored_during_execution":[{"topology_key":"zone","match_label_keys":[],"mismatch_label_keys":[],"label_selector":[{"match_labels":{"app":"test"}}]}]}]}]}]}`,
			"windows": `{"spec":[{"container":[{"name":"app","security_context":[{"windows_options":{"run_as_username":"ContainerUser","host_process":false}}]}],"init_container":[{"name":"init","security_context":[{"windows_options":{"gmsa_credential_spec_name":"domain"}}]}]}]}`,
		} {
			t.Run(name, func(t *testing.T) {
				s := schema.Schema{Blocks: map[string]schema.Block{"spec": b.Spec}}
				decoded, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(s.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: s, Raw: decoded}
				var baseline types.List
				if d := state.GetAttribute(ctx, path.Root("spec"), &baseline); d.HasError() {
					t.Fatal(d)
				}
				api, d := b.ExpandSpec(ctx, baseline, nil, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				if name == "linux" {
					c := api.Containers[0]
					if api.SetHostnameAsFQDN == nil || !*api.SetHostnameAsFQDN || api.SecurityContext.AppArmorProfile == nil || api.SecurityContext.SupplementalGroupsPolicy == nil || api.SecurityContext.SELinuxChangePolicy == nil || c.SecurityContext.AppArmorProfile == nil || c.LivenessProbe.TCPSocket.Host != "127.0.0.1" || c.LivenessProbe.TerminationGracePeriodSeconds == nil || c.StartupProbe.TerminationGracePeriodSeconds == nil || c.Lifecycle.PreStop.Sleep == nil || c.Lifecycle.PreStop.Sleep.Seconds != 0 || c.VolumeMounts[0].RecursiveReadOnly == nil || c.Ports[0].Protocol != corev1.ProtocolSCTP {
						t.Fatalf("features lost in API request: %#v", api)
					}
				} else if name == "windows" && (api.Containers[0].SecurityContext.WindowsOptions == nil || api.Containers[0].SecurityContext.WindowsOptions.HostProcess == nil || *api.Containers[0].SecurityContext.WindowsOptions.HostProcess || api.InitContainers[0].SecurityContext.WindowsOptions == nil) {
					t.Fatal("Windows inheritance or explicit false lost")
				}
				wire, err := json.Marshal(api)
				if err != nil {
					t.Fatal(err)
				}
				var observed corev1.PodSpec
				if err := json.Unmarshal(wire, &observed); err != nil {
					t.Fatal(err)
				}
				refreshed, d := b.RefreshSpec(ctx, observed, baseline, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				var check func(attr.Value, attr.Value, string)
				check = func(before, after attr.Value, key string) {
					if podOptionalFeaturePath(key) {
						if !before.Equal(after) {
							t.Errorf("%s changed across refresh: %s -> %s", key, before, after)
						}
						return
					}
					switch v := before.(type) {
					case types.Object:
						if v.IsNull() {
							return
						}
						for field, child := range v.Attributes() {
							check(child, after.(types.Object).Attributes()[field], key+"."+field)
						}
					case types.List:
						if len(v.Elements()) > len(after.(types.List).Elements()) {
							t.Fatalf("%s lost list elements after refresh", key)
						}
						for i, child := range v.Elements() {
							if i < len(after.(types.List).Elements()) {
								check(child, after.(types.List).Elements()[i], key)
							}
						}
					}
				}
				check(baseline, refreshed, "spec")
			})
		}
	}
}

func TestAffinityLabelKeysRefresh(t *testing.T) {
	ctx := context.Background()
	b := For(Pod())
	term := corev1.PodAffinityTerm{TopologyKey: "kubernetes.io/hostname", MatchLabelKeys: []string{"revision"}, MismatchLabelKeys: []string{"tenant"}, LabelSelector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "tier", Operator: metav1.LabelSelectorOpExists}, {Key: "tenant", Operator: metav1.LabelSelectorOpExists}}}}
	live := corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}, Affinity: &corev1.Affinity{
		PodAffinity:     &corev1.PodAffinity{RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{term}, PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{Weight: 1, PodAffinityTerm: term}}},
		PodAntiAffinity: &corev1.PodAntiAffinity{RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{term}, PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{Weight: 1, PodAffinityTerm: term}}},
	}}
	baseline, d := b.RefreshSpec(ctx, live, types.ListNull(b.ObjectType()), path.Root("spec"))
	if d.HasError() {
		t.Fatal(d)
	}
	live = *live.DeepCopy()
	for _, term := range podAffinityTerms(live.Affinity) {
		term.LabelSelector.MatchExpressions = append(term.LabelSelector.MatchExpressions,
			metav1.LabelSelectorRequirement{Key: "revision", Operator: metav1.LabelSelectorOpIn, Values: []string{"original-revision"}},
			metav1.LabelSelectorRequirement{Key: "tenant", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"original-tenant"}},
		)
	}
	refreshed, d := b.RefreshSpec(ctx, live, baseline, path.Root("spec"))
	if d.HasError() || !refreshed.Equal(baseline) {
		t.Fatalf("generated affinity expressions changed state: %s", d)
	}
	flattened, d := b.FlattenSpec(ctx, live, baseline, path.Root("spec"))
	if d.HasError() || !Satisfies(flattened, baseline) {
		t.Fatalf("generated affinity expressions require Pod replacement: %s", d)
	}
	for _, term := range podAffinityTerms(live.Affinity) {
		term.LabelSelector.MatchExpressions = append(term.LabelSelector.MatchExpressions,
			metav1.LabelSelectorRequirement{Key: "external", Operator: metav1.LabelSelectorOpIn, Values: []string{"admission"}})
	}
	for _, affinity := range []string{"pod_affinity", "pod_anti_affinity"} {
		for _, preferred := range []bool{false, true} {
			at := []interface{}{0, "affinity", 0, affinity, 0, "required_during_scheduling_ignored_during_execution", 0}
			if preferred {
				at[5] = "preferred_during_scheduling_ignored_during_execution"
				at = append(at, "pod_affinity_term", 0)
			}
			for _, owned := range []bool{true, false} {
				prior := baseline
				if !owned {
					for _, field := range []string{"match_label_keys", "mismatch_label_keys"} {
						prior = with(t, prior, types.SetNull(types.StringType), append(at, field)...).(types.List)
					}
				}
				refreshed, d := b.RefreshSpec(ctx, live, prior, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				expressions := get(refreshed, append(at, "label_selector", 0, "match_expressions")...).(types.List)
				want := 3
				if !owned {
					want = 5
				}
				if len(expressions.Elements()) != want {
					t.Fatalf("%s preferred=%t owned=%t: %s", affinity, preferred, owned, expressions)
				}
				if get(expressions, 0, "key").(types.String).ValueString() != "tier" {
					t.Fatal("configured selector expression removed")
				}
				if get(expressions, 1, "operator").(types.String).ValueString() != "Exists" {
					t.Fatal("configured same-key selector expression removed")
				}
			}
			// Preserve the exact configured predicates, including duplicates,
			// when admission appends a different or identical predicate.
			for _, tc := range []struct {
				value string
				count int
			}{{"configured-tenant", 1}, {"original-tenant", 1}, {"original-tenant", 2}} {
				expressionPath := append(at, "label_selector", 0, "match_expressions")
				expressions := get(baseline, expressionPath...).(types.List)
				predicate := with(t, expressions.Elements()[1], types.StringValue("NotIn"), "operator")
				predicate = with(t, predicate, types.SetValueMust(types.StringType, []attr.Value{types.StringValue(tc.value)}), "values")
				configured := expressions.Elements()
				withPriorPredicate := live.DeepCopy()
				for range tc.count {
					configured = append(configured, predicate)
					for _, term := range podAffinityTerms(withPriorPredicate.Affinity) {
						term.LabelSelector.MatchExpressions = slices.Insert(term.LabelSelector.MatchExpressions, 2,
							metav1.LabelSelectorRequirement{Key: "tenant", Operator: metav1.LabelSelectorOpNotIn, Values: []string{tc.value}})
					}
				}
				prior := with(t, baseline, types.ListValueMust(expressions.ElementType(ctx), configured), expressionPath...).(types.List)
				refreshed, d := b.RefreshSpec(ctx, *withPriorPredicate, prior, path.Root("spec"))
				if d.HasError() {
					t.Fatal(d)
				}
				got := get(refreshed, expressionPath...).(types.List)
				if len(got.Elements()) != 3+tc.count || !slices.EqualFunc(got.Elements()[:2+tc.count], configured, func(a, b attr.Value) bool { return a.Equal(b) }) {
					t.Fatalf("%s preferred=%t value=%s count=%d: configured predicate lost or generated predicate retained: %s", affinity, preferred, tc.value, tc.count, got)
				}
			}
		}
	}
}

func TestOptionalPodFeatureRuntimeDefaults(t *testing.T) {
	ctx := context.Background()
	b := For(Pod())
	api := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data", ReadOnly: true}}}}, SecurityContext: &corev1.PodSecurityContext{RunAsUser: ptr.To(int64(1000))}}
	baseline, d := b.RefreshSpec(ctx, api, types.ListNull(b.ObjectType()), path.Root("spec"))
	if d.HasError() {
		t.Fatal(d)
	}
	for _, tc := range []struct {
		at                       []interface{}
		defaultValue, nondefault attr.Value
	}{
		{[]interface{}{0, "set_hostname_as_fqdn"}, types.BoolValue(false), types.BoolValue(true)},
		{[]interface{}{0, "security_context", 0, "supplemental_groups_policy"}, types.StringValue("Merge"), types.StringValue("Strict")},
		{[]interface{}{0, "container", 0, "volume_mount", 0, "recursive_read_only"}, types.StringValue("Disabled"), types.StringValue("Enabled")},
	} {
		for _, value := range []attr.Value{tc.defaultValue, tc.nondefault} {
			prior := with(t, baseline, value, tc.at...).(types.List)
			wire, err := json.Marshal(api)
			if err != nil {
				t.Fatal(err)
			}
			var observed corev1.PodSpec
			if err := json.Unmarshal(wire, &observed); err != nil {
				t.Fatal(err)
			}
			refreshed, d := b.RefreshSpec(ctx, observed, prior, path.Root("spec"))
			if d.HasError() {
				t.Fatal(d)
			}
			got := get(refreshed, tc.at...)
			if value.Equal(tc.defaultValue) && !got.Equal(value) {
				t.Fatalf("runtime default lost: %v: %s -> %s", tc.at, value, got)
			}
			if value.Equal(tc.nondefault) && !got.IsNull() {
				t.Fatalf("API dropped nondefault without drift: %v: %s", tc.at, got)
			}
		}
	}
}
