// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

// SDKv2 state holds container resources with only zero values as [], the same
// object Kubernetes holds for a configured [{}].
func TestAbsentZeroListStructure(t *testing.T) {
	ctx := context.Background()
	element := types.ObjectType{AttrTypes: map[string]attr.Type{
		"limits":   types.MapType{ElemType: types.StringType},
		"requests": types.MapType{ElemType: types.StringType},
	}}
	resources := func(limits, requests types.Map) types.List {
		return types.ListValueMust(element, []attr.Value{types.ObjectValueMust(element.AttrTypes, map[string]attr.Value{"limits": limits, "requests": requests})})
	}
	null, unknown := types.MapNull(types.StringType), types.MapUnknown(types.StringType)
	empty := types.MapValueMust(types.StringType, map[string]attr.Value{})
	cpu := types.MapValueMust(types.StringType, map[string]attr.Value{"cpu": types.StringValue("100m")})
	for name, tc := range map[string]struct {
		modifier     podListStructureRequiresReplace
		config, plan types.List
		replace      bool
		want         types.List
	}{
		"zero element":       {podListStructureRequiresReplace{absentZero: true}, resources(null, null), resources(unknown, unknown), false, resources(empty, empty)},
		"limits set":         {podListStructureRequiresReplace{absentZero: true}, resources(cpu, null), resources(cpu, unknown), true, resources(cpu, unknown)},
		"other list":         {podListStructureRequiresReplace{}, resources(null, null), resources(unknown, unknown), true, resources(unknown, unknown)},
		"unknown configured": {podListStructureRequiresReplace{absentZero: true}, resources(unknown, null), resources(unknown, unknown), true, resources(unknown, unknown)},
	} {
		req := planmodifier.ListRequest{StateValue: types.ListValueMust(element, nil), ConfigValue: tc.config, PlanValue: tc.plan}
		req.State.Raw = tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
		req.Plan.Raw = req.State.Raw
		resp := planmodifier.ListResponse{PlanValue: tc.plan}
		tc.modifier.PlanModifyList(ctx, req, &resp)
		if resp.RequiresReplace != tc.replace || !resp.PlanValue.Equal(tc.want) {
			t.Errorf("%s: replace = %t, plan = %s; want %t, %s", name, resp.RequiresReplace, resp.PlanValue, tc.replace, tc.want)
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

// Adding a pod-spec element replaces a workload only when it holds a value that
// forces replacement for that owner. As in SDKv2, an unknown value, such as a
// dynamic block over a value known only after apply, replaces only when the
// unknown node is itself ForceNew.
func TestAddedElementReplacement(t *testing.T) {
	ctx := context.Background()
	object := func(typ attr.Type, set map[string]attr.Value) types.Object {
		t := typ.(types.ObjectType)
		values := map[string]attr.Value{}
		for name, typ := range t.AttrTypes {
			if v, ok := set[name]; ok {
				values[name] = v
				continue
			}
			values[name], _ = typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
		}
		return types.ObjectValueMust(t.AttrTypes, values)
	}
	elem := func(parent attr.Type, name string) attr.Type {
		return parent.(types.ObjectType).AttrTypes[name].(types.ListType).ElemType
	}
	one := func(typ attr.Type, set map[string]attr.Value) types.List {
		return types.ListValueMust(typ, []attr.Value{object(typ, set)})
	}
	volume := func(b builder) schema.NestedBlockObject { return b.podVolumeObject() }
	affinity := func(b builder) schema.NestedBlockObject { return b.podAffinityObject() }
	// nodeTerms builds node_affinity.required_during_scheduling_ignored_during_execution.
	nodeTerms := func(a attr.Type, terms func(term attr.Type) types.List) map[string]attr.Value {
		node := elem(a, "node_affinity")
		required := elem(node, "required_during_scheduling_ignored_during_execution")
		return map[string]attr.Value{"node_affinity": one(node, map[string]attr.Value{
			"required_during_scheduling_ignored_during_execution": one(required, map[string]attr.Value{
				"node_selector_term": terms(elem(required, "node_selector_term")),
			}),
		})}
	}
	for name, tc := range map[string]struct {
		options Options
		object  func(builder) schema.NestedBlockObject
		element func(typ attr.Type) map[string]attr.Value
		replace bool
	}{
		"config_map with unknown items": {DaemonSet(), volume, func(v attr.Type) map[string]attr.Value {
			cm := elem(v, "config_map")
			return map[string]attr.Value{"name": types.StringValue("v"), "config_map": one(cm, map[string]attr.Value{
				"name": types.StringValue("cm"), "items": types.ListUnknown(elem(cm, "items")),
			})}
		}, false},
		"unknown azure_file": {DaemonSet(), volume, func(v attr.Type) map[string]attr.Value {
			return map[string]attr.Value{"name": types.StringValue("v"), "azure_file": types.ListUnknown(elem(v, "azure_file"))}
		}, false},
		"unknown secret_namespace": {DaemonSet(), volume, func(v attr.Type) map[string]attr.Value {
			return map[string]attr.Value{"name": types.StringValue("v"), "azure_file": one(elem(v, "azure_file"), map[string]attr.Value{
				"secret_name": types.StringValue("s"), "share_name": types.StringValue("s"), "secret_namespace": types.StringUnknown(),
			})}
		}, true},
		"unknown empty_dir on a template": {DaemonSet(), volume, func(v attr.Type) map[string]attr.Value {
			return map[string]attr.Value{"name": types.StringValue("v"), "empty_dir": types.ListUnknown(elem(v, "empty_dir"))}
		}, false},
		"unknown node_selector_term": {Deployment(), affinity, func(a attr.Type) map[string]attr.Value {
			return nodeTerms(a, types.ListUnknown)
		}, false},
		"unknown match_fields": {Deployment(), affinity, func(a attr.Type) map[string]attr.Value {
			return nodeTerms(a, func(term attr.Type) types.List {
				return one(term, map[string]attr.Value{"match_fields": types.ListUnknown(elem(term, "match_fields"))})
			})
		}, true},
		"known match_expressions": {Deployment(), affinity, func(a attr.Type) map[string]attr.Value {
			return nodeTerms(a, func(term attr.Type) types.List {
				expression := elem(term, "match_expressions")
				return one(term, map[string]attr.Value{"match_expressions": one(expression, map[string]attr.Value{
					"key": types.StringValue("k"), "operator": types.StringValue("Exists"),
				})})
			})
		}, false},
	} {
		t.Run(name, func(t *testing.T) {
			parent := tc.object(builder{o: tc.options})
			typ := parent.Type()
			req := planmodifier.ListRequest{
				StateValue: types.ListValueMust(typ, nil),
				PlanValue:  types.ListValueMust(typ, []attr.Value{object(typ, tc.element(typ))}),
			}
			req.State.Raw = tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
			req.Plan.Raw = req.State.Raw
			var resp planmodifier.ListResponse
			podListInheritedRequiresReplace{object: parent}.PlanModifyList(ctx, req, &resp)
			if resp.RequiresReplace != tc.replace {
				t.Errorf("replace = %t, want %t", resp.RequiresReplace, tc.replace)
			}
		})
	}
}

// As in SDKv2, a replacing collection that is unknown until apply replaces.
func TestCollectionSizeReplacement(t *testing.T) {
	element := types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType}}
	list := func(n int) types.List {
		elements := make([]attr.Value, n)
		for i := range elements {
			elements[i] = types.ObjectValueMust(element.AttrTypes, map[string]attr.Value{"key": types.StringValue("k")})
		}
		return types.ListValueMust(element, elements)
	}
	for name, tc := range map[string]struct {
		state, plan types.List
		replace     bool
	}{
		"same size":     {list(1), list(1), false},
		"added":         {list(0), list(1), true},
		"removed":       {list(1), types.ListNull(element), true},
		"unknown":       {list(1), types.ListUnknown(element), true},
		"unknown prior": {types.ListUnknown(element), list(1), false},
	} {
		t.Run(name, func(t *testing.T) {
			req := planmodifier.ListRequest{StateValue: tc.state, PlanValue: tc.plan}
			req.State.Raw = tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
			req.Plan.Raw = req.State.Raw
			var resp planmodifier.ListResponse
			podListStructureRequiresReplace{}.PlanModifyList(context.Background(), req, &resp)
			if resp.RequiresReplace != tc.replace {
				t.Errorf("replace = %t, want %t", resp.RequiresReplace, tc.replace)
			}
		})
	}
}

// Strings Kubernetes stores as numbers keep their configured spelling.
func TestNumberSpellingPaths(t *testing.T) {
	b := For(Pod())
	for _, key := range []string{
		"spec.security_context.run_as_user",
		"spec.container.security_context.run_as_group",
		"spec.toleration.toleration_seconds",
		"spec.container.liveness_probe.http_get.port",
		"spec.volume.secret.default_mode",
		"spec.volume.config_map.items.mode",
	} {
		if b.spelling[key] == nil {
			t.Errorf("%s: spelling not kept", key)
		}
	}
	if b.spelling["spec.container.image"] != nil {
		t.Error("spec.container.image: spelling kept")
	}
}

// As in SDKv2, a pod security_context holding only zero values is sent with
// runAsNonRoot = false when its configuration sets a bool or a nested block.
// Without a configuration it is unset, as plan predictions compare it.
func TestZeroSecurityContextPayload(t *testing.T) {
	ctx := context.Background()
	b := For(Pod())
	at := path.Root("spec")
	// block is one element of typ with every attribute null except set; a nil
	// value in set stands for one such element of a nested block.
	var block func(typ attr.Type, set map[string]attr.Value) types.List
	block = func(typ attr.Type, set map[string]attr.Value) types.List {
		object := typ.(types.ObjectType)
		values := map[string]attr.Value{}
		for name, child := range object.AttrTypes {
			values[name], _ = child.ValueFromTerraform(ctx, tftypes.NewValue(child.TerraformType(ctx), nil))
			if value, ok := set[name]; ok && value == nil {
				values[name] = block(child.(types.ListType).ElemType, nil)
			} else if ok {
				values[name] = value
			}
		}
		return types.ListValueMust(object, []attr.Value{types.ObjectValueMust(object.AttrTypes, values)})
	}
	zero := corev1.PodSecurityContext{RunAsNonRoot: ptr.To(false)}
	noGroups := types.SetValueMust(types.Int64Type, nil)
	for _, tc := range []struct {
		name   string
		live   corev1.PodSecurityContext
		config map[string]attr.Value // nil: no configuration
		want   *bool
	}{
		{"empty block", zero, map[string]attr.Value{}, nil},
		{"empty supplemental groups", zero, map[string]attr.Value{"supplemental_groups": noGroups}, nil},
		{"empty run_as_user", zero, map[string]attr.Value{"run_as_user": types.StringValue("")}, nil},
		{"run_as_non_root false", zero, map[string]attr.Value{"run_as_non_root": types.BoolValue(false)}, ptr.To(false)},
		{"false with empty groups", zero, map[string]attr.Value{"run_as_non_root": types.BoolValue(false), "supplemental_groups": noGroups}, ptr.To(false)},
		{"empty se_linux_options", zero, map[string]attr.Value{"se_linux_options": nil}, ptr.To(false)},
		{"run_as_non_root true", corev1.PodSecurityContext{RunAsNonRoot: ptr.To(true)}, map[string]attr.Value{"run_as_non_root": types.BoolValue(true)}, ptr.To(true)},
		{"run_as_user set", corev1.PodSecurityContext{RunAsUser: ptr.To(int64(1000))}, map[string]attr.Value{"run_as_user": types.StringValue("1000")}, ptr.To(false)},
		{"no configuration", zero, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "pause"}}, SecurityContext: &tc.live}
			plan, diags := b.FlattenSpec(ctx, live, types.ListNull(b.ObjectType()), at)
			if diags.HasError() {
				t.Fatal(diags)
			}
			var config *tfsdk.Config
			if tc.config != nil {
				spec := plan.Elements()[0].(types.Object)
				attributes := spec.Attributes()
				attributes["security_context"] = block(attributes["security_context"].(types.List).ElementType(ctx), tc.config)
				configured := types.ListValueMust(plan.ElementType(ctx), []attr.Value{types.ObjectValueMust(spec.AttributeTypes(ctx), attributes)})
				raw, err := types.ObjectValueMust(map[string]attr.Type{"spec": plan.Type(ctx)}, map[string]attr.Value{"spec": configured}).ToTerraformValue(ctx)
				if err != nil {
					t.Fatal(err)
				}
				config = &tfsdk.Config{Schema: schema.Schema{Blocks: map[string]schema.Block{"spec": b.Spec}}, Raw: raw}
			}
			got, diags := b.ExpandSpec(ctx, plan, config, at)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !reflect.DeepEqual(got.SecurityContext.RunAsNonRoot, tc.want) {
				t.Errorf("runAsNonRoot = %v, want %v", got.SecurityContext.RunAsNonRoot, tc.want)
			}
		})
	}
}
