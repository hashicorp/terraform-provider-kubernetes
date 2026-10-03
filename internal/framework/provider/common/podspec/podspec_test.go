// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"context"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

// Each owner forces replacement exactly where its SDKv2 resource declared ForceNew.
// SDKv2 is read through its deprecated unversioned names, the ones it still
// registers; kubernetes_cron_job is batch/v1beta1, whose job template is the
// same jobSpecFields schema as the v1 resource.
func TestReplacementMatchesSDKv2(t *testing.T) {
	sdk := kubernetes.Provider().ResourcesMap
	template := []string{"spec", "template", "spec"}
	for name, tc := range map[string]struct {
		options Options
		sdk     string
		at      []string
	}{
		"deployment":   {Deployment(), "kubernetes_deployment", template},
		"daemon_set":   {DaemonSet(), "kubernetes_daemonset", template},
		"stateful_set": {StatefulSet(), "kubernetes_stateful_set", template},
		"pod":          {Pod(), "kubernetes_pod", []string{"spec"}},
		"job":          {Job(), "kubernetes_job", template},
		"cron_job":     {CronJob(), "kubernetes_cron_job", []string{"spec", "job_template", "spec", "template", "spec"}},
	} {
		t.Run(name, func(t *testing.T) {
			legacy := sdk[tc.sdk].Schema[tc.at[0]]
			for _, step := range tc.at[1:] {
				legacy = legacy.Elem.(*sdkschema.Resource).Schema[step]
			}
			want := map[string]bool{}
			sdkForceNew("spec", legacy, want)
			if tc.options.Immutable {
				// SDKv2 omitted ForceNew here, but the API rejects the change.
				want["spec.dns_config.searches"] = true
			}

			spec := For(tc.options).Spec
			got := map[string]bool{"spec": hasReplacement(spec.PlanModifiers)}
			frameworkReplacement("spec", spec.NestedObject.Attributes, spec.NestedObject.Blocks, got)
			for path := range union(want, got) {
				if want[path] != got[path] {
					t.Errorf("%s: replacement = %t, SDKv2 ForceNew = %t", path, got[path], want[path])
				}
			}
			var restartPolicy defaults.StringResponse
			spec.NestedObject.Attributes["restart_policy"].(schema.StringAttribute).Default.DefaultString(context.Background(), defaults.StringRequest{}, &restartPolicy)
			if want := legacy.Elem.(*sdkschema.Resource).Schema["restart_policy"].Default; restartPolicy.PlanValue.ValueString() != want {
				t.Errorf("restart_policy default = %s, SDKv2 = %v", restartPolicy.PlanValue, want)
			}
		})
	}
}

func sdkForceNew(path string, s *sdkschema.Schema, out map[string]bool) {
	out[path] = s.ForceNew
	if r, ok := s.Elem.(*sdkschema.Resource); ok {
		for name, child := range r.Schema {
			sdkForceNew(path+"."+name, child, out)
		}
	}
}

// Lists of objects that SDKv2 modelled as blocks carry their children's
// replacement on the list itself.
func frameworkReplacement(path string, attributes map[string]schema.Attribute, blocks map[string]schema.Block, out map[string]bool) {
	for name, attribute := range attributes {
		p := path + "." + name
		switch a := attribute.(type) {
		case schema.ListAttribute:
			object, ok := a.ElementType.(types.ObjectType)
			if !ok {
				out[p] = hasReplacement(a.PlanModifiers)
				continue
			}
			out[p] = false
			for child := range object.AttrTypes {
				out[p+"."+child] = hasReplacement(a.PlanModifiers)
			}
		case schema.ListNestedAttribute:
			out[p] = hasReplacement(a.PlanModifiers)
			frameworkReplacement(p, a.NestedObject.Attributes, nil, out)
		default:
			out[p] = podObjectRequiresStructuralReplacement(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{name: attribute}})
		}
	}
	for name, block := range blocks {
		b := block.(schema.ListNestedBlock)
		p := path + "." + name
		out[p] = slices.ContainsFunc(b.PlanModifiers, func(m planmodifier.List) bool {
			_, structural := m.(podListStructureRequiresReplace)
			return structural
		})
		frameworkReplacement(p, b.NestedObject.Attributes, b.NestedObject.Blocks, out)
	}
}

func hasReplacement(modifiers []planmodifier.List) bool {
	return slices.ContainsFunc(modifiers, func(m planmodifier.List) bool {
		_, inherited := m.(podListInheritedRequiresReplace)
		return !inherited && podModifiersRequireReplacement([]planmodifier.List{m})
	})
}

func union(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

// SDKv2 stored omitted scalars as zero values, so moving between zero and
// null must not replace an immutable pod spec.
func TestReplacementIgnoresZeroValues(t *testing.T) {
	modifier := podStringRequiresReplace{stringplanmodifier.RequiresReplace()}
	for _, tc := range []struct {
		state, plan types.String
		replace     bool
	}{
		{types.StringValue(""), types.StringNull(), false},
		{types.StringNull(), types.StringValue(""), false},
		{types.StringValue("a"), types.StringNull(), true},
		{types.StringValue("a"), types.StringValue("b"), true},
	} {
		req := planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan, ConfigValue: tc.plan}
		req.State.Raw = tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
		req.Plan.Raw = req.State.Raw
		var resp planmodifier.StringResponse
		modifier.PlanModifyString(context.Background(), req, &resp)
		if resp.RequiresReplace != tc.replace {
			t.Errorf("%s -> %s: replace = %t, want %t", tc.state, tc.plan, resp.RequiresReplace, tc.replace)
		}
	}
}

// Core accepts a planned map equal to the configuration or the prior state,
// never a mixture, so respelled quantities are kept only all at once.
func TestQuantityMapKeepsPriorSpellingWholesale(t *testing.T) {
	quantities := func(kv ...string) types.Map {
		values := map[string]attr.Value{}
		for i := 0; i < len(kv); i += 2 {
			values[kv[i]] = types.StringValue(kv[i+1])
		}
		return types.MapValueMust(types.StringType, values)
	}
	prior := quantities("cpu", "500m", "memory", "1Gi")
	for name, tc := range map[string]struct {
		plan, want types.Map
	}{
		"all equivalent":  {quantities("cpu", "0.5", "memory", "1024Mi"), prior},
		"one changed":     {quantities("cpu", "0.5", "memory", "2Gi"), quantities("cpu", "0.5", "memory", "2Gi")},
		"key added":       {quantities("cpu", "0.5", "memory", "1Gi", "storage", "1Gi"), quantities("cpu", "0.5", "memory", "1Gi", "storage", "1Gi")},
		"key removed":     {quantities("cpu", "0.5"), quantities("cpu", "0.5")},
		"identical":       {prior, prior},
		"null configured": {types.MapNull(types.StringType), types.MapNull(types.StringType)},
	} {
		t.Run(name, func(t *testing.T) {
			resp := planmodifier.MapResponse{PlanValue: tc.plan}
			podQuantityMapPlanModifier{}.PlanModifyMap(context.Background(), planmodifier.MapRequest{StateValue: prior, PlanValue: tc.plan}, &resp)
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("planned %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

// "" and null both mean "unset" for an API-defaulted string. A write keeps the
// planned "" so the result matches the plan; a read records the live value.
func TestKeepUnsetString(t *testing.T) {
	for _, tc := range []struct {
		baseline types.String
		api      string
		refresh  bool
		keep     bool
	}{
		{types.StringValue(""), "IfNotPresent", false, true},
		{types.StringValue(""), "IfNotPresent", true, false},
		{types.StringValue(""), "", false, true},
		{types.StringNull(), "", false, true},
		{types.StringNull(), "", true, true},
		{types.StringNull(), "Always", true, false},
		{types.StringValue("Always"), "IfNotPresent", false, false},
		{types.StringUnknown(), "IfNotPresent", false, false},
	} {
		if got := podKeepUnsetString(tc.baseline, tc.api, tc.refresh); got != tc.keep {
			t.Errorf("baseline %s, API %q, refresh %t: keep = %t, want %t", tc.baseline, tc.api, tc.refresh, got, tc.keep)
		}
	}
}

// A configured block holding only zero values is kept when Kubernetes returns
// none: always after a write, and on a read only where Kubernetes cannot hold
// such a block, so removing one out of band shows as drift.
func TestZeroValueBlockReadBack(t *testing.T) {
	ctx := context.Background()
	b := For(Deployment())
	at := path.Root("spec")
	live := corev1.PodSpec{Containers: []corev1.Container{{
		Name:            "app",
		Image:           "nginx",
		SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: ptr.To(false)},
	}}}
	baseline, diags := b.FlattenSpec(ctx, live, types.ListNull(b.ObjectType()), at)
	if diags.HasError() {
		t.Fatal(diags)
	}
	live.Containers[0].SecurityContext = nil
	for _, tc := range []struct {
		name    string
		flatten func(context.Context, corev1.PodSpec, types.List, path.Path) (types.List, diag.Diagnostics)
		kept    bool
	}{
		{"write", b.FlattenSpec, true},
		{"read", b.RefreshSpec, false},
	} {
		got, diags := tc.flatten(ctx, live, baseline, at)
		if diags.HasError() {
			t.Fatal(diags)
		}
		container := got.Elements()[0].(types.Object).Attributes()["container"].(types.List).Elements()[0]
		kept := len(container.(types.Object).Attributes()["security_context"].(types.List).Elements()) == 1
		if kept != tc.kept {
			t.Errorf("%s: container security_context kept = %t, want %t", tc.name, kept, tc.kept)
		}
	}
	for key, want := range map[string]bool{
		"spec.security_context":           true,
		"spec.dns_config":                 true,
		"spec.container.security_context": false,
		"spec.container.resources":        false,
	} {
		if b.zeroAbsent[key] != want {
			t.Errorf("%s: zero block equivalent to absent = %t, want %t", key, b.zeroAbsent[key], want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	block := types.ObjectType{AttrTypes: map[string]attr.Type{"value": types.BoolType}}
	list := func(values ...attr.Value) types.List {
		elements := make([]attr.Value, len(values))
		for i, value := range values {
			elements[i] = types.ObjectValueMust(block.AttrTypes, map[string]attr.Value{"value": value})
		}
		return types.ListValueMust(block, elements)
	}
	for name, test := range map[string]struct {
		actual, planned attr.Value
		want            bool
	}{
		"equal":                  {list(types.BoolValue(false)), list(types.BoolValue(false)), true},
		"empty matches null":     {list(), types.ListNull(block), true},
		"unknown matches value":  {list(types.BoolValue(true)), list(types.BoolUnknown()), true},
		"different value":        {list(types.BoolValue(true)), list(types.BoolValue(false)), false},
		"block the plan removes": {list(types.BoolValue(false)), list(), false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := Satisfies(test.actual, test.planned); got != test.want {
				t.Fatalf("Satisfies = %t, want %t", got, test.want)
			}
		})
	}
}

// SDKv2 recorded a block Kubernetes always holds, such as container resources,
// as an empty list when it held only zero values. A write keeps that list; a
// read records the live block.
func TestAbsentZeroBlockWriteBack(t *testing.T) {
	ctx := context.Background()
	b := For(Job())
	at := path.Root("spec")
	live := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}}
	full, diags := b.FlattenSpec(ctx, live, types.ListNull(b.ObjectType()), at)
	if diags.HasError() {
		t.Fatal(diags)
	}
	spec := full.Elements()[0].(types.Object)
	containers := spec.Attributes()["container"].(types.List)
	container := containers.Elements()[0].(types.Object)
	resources := container.Attributes()["resources"].(types.List)
	with := func(object types.Object, name string, value attr.Value) types.Object {
		attributes := object.Attributes()
		attributes[name] = value
		return types.ObjectValueMust(object.AttributeTypes(ctx), attributes)
	}
	container = with(container, "resources", types.ListValueMust(resources.ElementType(ctx), nil))
	spec = with(spec, "container", types.ListValueMust(containers.ElementType(ctx), []attr.Value{container}))
	baseline := types.ListValueMust(full.ElementType(ctx), []attr.Value{spec})

	limited := *live.DeepCopy()
	limited.Containers[0].Resources.Limits = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")}
	for _, tc := range []struct {
		name    string
		flatten func(context.Context, corev1.PodSpec, types.List, path.Path) (types.List, diag.Diagnostics)
		live    corev1.PodSpec
		want    bool
	}{
		{"write", b.FlattenSpec, live, true},
		{"read", b.RefreshSpec, live, false},
		{"write with limits", b.FlattenSpec, limited, false},
	} {
		got, diags := tc.flatten(ctx, tc.live, baseline, at)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if Satisfies(got, baseline) != tc.want {
			t.Errorf("%s: satisfies a planned empty resources list = %t, want %t", tc.name, !tc.want, tc.want)
		}
	}
	for key, want := range map[string]bool{
		"spec.container.resources":        true,
		"spec.init_container.resources":   true,
		"spec.security_context":           false,
		"spec.container.security_context": false,
	} {
		if b.absentZero[key] != want {
			t.Errorf("%s: absent block held as zero = %t, want %t", key, b.absentZero[key], want)
		}
	}
}
